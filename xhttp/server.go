// Package xhttp extends net/http with a few independent and opt-in
// production-ready abstractions built on stdlib principles, being simple,
// focused, and composable.
//
// Current extensions:
//   - [Server] wraps [http.Server] with signal-based graceful shutdown, lifecycle hooks, and sensible defaults
//   - [Err] type and [Error] function for separating internal errors from client-safe messages in HTTP handlers
//
// [Server] usage:
//
//	srv, err := xhttp.NewServer(&xhttp.ServerConfig{
//		UseTLS:      true,
//		TLSCertPath: "./cert.pem",
//		TLSKeyPath:  "./key.pem",
//		Handler:     myHandler,
//		AfterListen: func() { /* do something after listen */ },
//		AfterListenDelay: 2 * time.Second, // delay before calling AfterListen, defaults to 1 second
//		OnShutdown:  func() { /* cleanup database connections, websockets, etc. */ },
//		// See [ServerConfig] for all options and defaults.
//	})
//	if err != nil {
//		log.Fatalf("failed to create server: %v", err)
//	}
//	log.Fatal(srv.Listen())
//
// [Err] and [Error] usage:
//
//	func SubFunc() error {
//		// do something that might fail with sensitive info in the error
//		_, err := sensitiveFoo()
//		if err != nil {
//			return &xhttp.Err{Code: 500, Msg: "An error occurred doing foo", Err: err}
//		}
//		return nil
//	}
//
//	func HandlerFunc(w http.ResponseWriter, r *http.Request) {
//		ctx := r.Context() // should contain github.com/Data-Corruption/stdx/xlog logger, skips logging if not present
//		if err := SubFunc(); err != nil {
//			// use [Error] instead of [http.Error]. It logs the error and sends an
//			// appropriate HTTP response, defaulting to 500, "Internal Server Error". If not an [Err].
//			xhttp.Error(ctx, w, err)
//			return
//		}
//		// continue handling the request
//	}
package xhttp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Default values for config, everything else defaults to zero values.
const (
	DefaultAddr             = ":80"
	DefaultTLSAddr          = ":443"
	DefaultReadTimeout      = 5 * time.Second
	DefaultWriteTimeout     = 10 * time.Second
	DefaultIdleTimeout      = 120 * time.Second
	DefaultShutdownTimeout  = 10 * time.Second
	DefaultAfterListenDelay = 1 * time.Second
)

// ServerConfig holds configuration options for [Server].
type ServerConfig struct {
	Addr string // Address to listen on (e.g. ":8080"). Default is ":80", ":443" if UseTLS is true.

	UseTLS      bool   // Whether to use TLS (HTTPS). If true, TLSKeyPath and TLSCertPath must be set.
	TLSKeyPath  string // Path to the TLS private key file.
	TLSCertPath string // Path to the TLS certificate file.

	// Handler, typically a router or middleware chain. Required.
	//
	// Works with any http.Handler compatible router (chi, gorilla/mux, etc.)
	Handler http.Handler

	ReadTimeout  time.Duration // Max duration for reading the entire request, including the body. Default is 5 seconds. Negative to disable.
	WriteTimeout time.Duration // Max duration before timing out writes of the response. Default is 10 seconds. Negative to disable.

	// IdleTimeout is the maximum duration for which an idle connection will remain open.
	// In plain terms, [http.Server] leaves connections open for a certain time after the last request
	// for performance reasons. This is the maximum duration for that. Default is 120 seconds. Negative to disable.
	//
	// This does not affect:
	//  - WebSocket connections (once upgraded)
	//  - Active request/response handling
	//  - Long-lived streaming responses (like SSE or chunked transfer)
	IdleTimeout time.Duration

	ShutdownTimeout time.Duration // Maximum duration for graceful shutdown. Default is 10 seconds. Zero or negative to disable.

	// AfterListen, if non-nil, is called after the server starts listening. Simple and flexible.
	// Useful for validating the server is up and running, e.g. by checking a health endpoint.
	AfterListen      func()
	AfterListenDelay time.Duration // Delay after starting the server before calling AfterListen.

	// OnShutdown, if non-nil, is called during server shutdown, after the
	// server has stopped accepting new connections, but before closing idle ones.
	//
	// Notes:
	//  - depending on the shutdown timeout, this may exceed the life of the server.
	//  - if ShutdownTimeout is <= 0, this will not be called.
	OnShutdown func()
}

// Server wraps [http.Server] with graceful shutdown, lifecycle hooks, and sensible defaults.
type Server struct {
	cfg    *ServerConfig // Configuration for the server
	server *http.Server  // The http or https server
}

// NewServer creates a new Server instance with the provided configuration.
func NewServer(cfg *ServerConfig) (*Server, error) {
	copy := *cfg

	if copy.Handler == nil {
		return nil, fmt.Errorf("handler must be provided")
	}

	if copy.UseTLS && (copy.TLSKeyPath == "" || copy.TLSCertPath == "") {
		return nil, fmt.Errorf("TLS key and cert paths must be provided when using TLS")
	}

	// set defaults

	if copy.Addr == "" {
		if copy.UseTLS {
			copy.Addr = DefaultTLSAddr
		} else {
			copy.Addr = DefaultAddr
		}
	}

	if copy.ReadTimeout == 0 {
		copy.ReadTimeout = DefaultReadTimeout
	}
	if copy.WriteTimeout == 0 {
		copy.WriteTimeout = DefaultWriteTimeout
	}
	if copy.IdleTimeout == 0 {
		copy.IdleTimeout = DefaultIdleTimeout
	}
	if copy.ShutdownTimeout == 0 {
		copy.ShutdownTimeout = DefaultShutdownTimeout
	}
	if copy.AfterListenDelay == 0 {
		copy.AfterListenDelay = DefaultAfterListenDelay
	}

	// create http server
	httpServer := &http.Server{
		Addr:         copy.Addr,
		Handler:      copy.Handler,
		ReadTimeout:  copy.ReadTimeout,
		WriteTimeout: copy.WriteTimeout,
		IdleTimeout:  copy.IdleTimeout,
		TLSConfig:    &tls.Config{MinVersion: tls.VersionTLS13},
	}

	// set shutdown hook if provided
	if copy.OnShutdown != nil && copy.ShutdownTimeout > 0 {
		httpServer.RegisterOnShutdown(copy.OnShutdown)
	}

	// return the server
	return &Server{
		cfg:    &copy,
		server: httpServer,
	}, nil
}

// Listen starts the server and blocks until it is shut down or an error occurs.
func (s *Server) Listen() error {
	// setup chans for listen and shutdown signals
	listenErrCh := make(chan error, 1)
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, os.Interrupt, syscall.SIGTERM)

	// start server
	go func() {
		if s.cfg.UseTLS {
			listenErrCh <- s.server.ListenAndServeTLS(s.cfg.TLSCertPath, s.cfg.TLSKeyPath)
		} else {
			listenErrCh <- s.server.ListenAndServe()
		}
	}()

	// setup AfterListen. For those curious, this is provided instead of OnListen as there is no way
	// to properly do OnListen with Go's http.Server. The closest would be polling. This is better.
	afterListenCh := make(chan struct{}, 1)
	if s.cfg.AfterListen != nil {
		go func() {
			time.Sleep(s.cfg.AfterListenDelay)
			afterListenCh <- struct{}{}
		}()
	}

	// handle AfterListen, shutdown, and listen errors
	for {
		select {
		case <-afterListenCh:
			s.cfg.AfterListen()
		case <-shutdownCh:
			signal.Stop(shutdownCh)
			if s.cfg.ShutdownTimeout <= 0 {
				return s.server.Close()
			}
			ctx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
			defer cancel()
			return s.server.Shutdown(ctx) // shutdown causes listen to return [ErrServerClosed] immediately, no need to handle it.
		case err := <-listenErrCh:
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				if errors.Is(err, syscall.EADDRINUSE) {
					return fmt.Errorf("address already in use: %w", err)
				}
				if errors.Is(err, syscall.EACCES) {
					return fmt.Errorf("permission denied: %w", err)
				}
				return err
			}
			return nil
		}
	}
}
