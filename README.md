# 🌰 stdx [![Go Reference](https://pkg.go.dev/badge/github.com/Data-Corruption/stdx.svg)](https://pkg.go.dev/github.com/Data-Corruption/stdx) [![Go Report Card](https://goreportcard.com/badge/github.com/Data-Corruption/stdx)](https://goreportcard.com/report/github.com/Data-Corruption/stdx) ![License](https://img.shields.io/github/license/Data-Corruption/stdx) [![Release](https://github.com/Data-Corruption/stdx/actions/workflows/release.yml/badge.svg)](https://github.com/Data-Corruption/stdx/actions/workflows/release.yml)

Production-hardened extensions for Go's standard library. Built with a bias toward CLI tools doing web-adjacent work (standalone apps, APIs, wrappers, etc). It's the result from building many small, durable CLI apps. Just enough structure to ship fast, fail loudly, and stay maintainable under pressure. No dependencies. No magic. Just practical helpers for real use.

- [`xhttp`](#xhttp): Production-ready HTTP server helpers
- [`xlog`](#xlog): Structured leveled logging
- [`xlog/rlog`](#xlogrlog): Buffered writer with rotation
- [`xnet`](#xlogrlog): Miscellaneous network helpers.
- [`xterm/prompt`](#xtermprompt): Interactive terminal prompts

## Installation

```bash
go get github.com/Data-Corruption/stdx
```

<br>

### xhttp

Package xhttp provides extensions to the standard net/http package for production-ready HTTP server functionality.

#### Features

- **`Server`**  
  A wrapper around `http.Server` that provides:
  - Signal-based graceful shutdown
  - Lifecycle hooks for actions after listening and before shutdown
  - Sensible defaults for server configuration
- **`Err`**  
  A custom error type for HTTP handlers that separates internal errors from client-safe messages.
- **`Error(ctx context.Context, w http.ResponseWriter, err error)`**  
  A function to handle errors in HTTP handlers, logging them and sending appropriate HTTP responses. A drop-in replacement for `http.Error` that works with the `xlog` logger in the context if present.

#### Quick example

```go
// Visit http://localhost:8080 to see a success response or, ~50% of the time,
// an internal error handled by xhttp.Error. Press Ctrl‑C to trigger a graceful
// shutdown.
package main

import (
  "errors"
  "log"
  "math/rand"
  "net/http"
  "time"

  "github.com/Data-Corruption/stdx/xhttp"
)

func main() {
  // Create router
  mux := http.NewServeMux()
  mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    if err := riskyOperation(); err != nil {
      // Logs then sends either the safe message or "Internal Server Error" if
      // not an xhttp.Err. Will also use xlog Logger if present in context
      xhttp.Error(r.Context(), w, err)
      return
    }
    w.Write([]byte("OK\n"))
  })

  // Create server
  var srv *xhttp.Server
  var err error
  srv, err = xhttp.NewServer(&xhttp.ServerConfig{
    Addr:    ":8080",
    Handler: mux,
    AfterListen: func() {
      log.Printf("server is ready and listening on http://localhost%s", srv.Addr())
    },
    OnShutdown: func() {
      log.Println("shutting down, cleaning up resources ...")
    },
    // See xhttp.ServerConfig for all options and defaults.
  })
  if err != nil {
    log.Fatalf("failed to create server: %v", err)
  }

  // Start serving (blocks until exit signal or error).
  if err := srv.Listen(); err != nil {
    log.Printf("server stopped with error: %v", err)
  } else {
    log.Println("server stopped gracefully")
  }
}

// riskyOperation simulates work that can fail.
func riskyOperation() error {
  if rand.Intn(2) == 0 {
    return &xhttp.Err{
      Code: http.StatusInternalServerError,
      Msg:  "Something went wrong, please try again later", // safe for clients
      Err:  errors.New("simulated failure"),                // internal detail
    }
  }
  time.Sleep(100 * time.Millisecond)
  return nil
}
```

<br>

### xlog

Package xlog provides a leveled, concurrent-safe logger with buffered rotation for logging.

#### Features

- **`Logger`**  
  A leveled logger that supports dynamic log level changes, custom formatting, and safe shutdown. Internally uses Writer from `xlog/rlog`.

#### Quick example

```go
package main

import (
  "log"

  "github.com/Data-Corruption/stdx/xlog"
)

func main() {
  // Create a new logger with debug level
  // All levels: "debug", "info", "warn", "error", "none"
  logger, err := xlog.New("./logs", "debug")
  if err != nil {
    log.Fatalf("xlog: %v", err)
  }
  defer logger.Close() // Ensure logs are flushed on exit

  // Log messages at different levels
  logger.Debug("Debugging information")
  logger.Infof("Formatted info: %s", "some_value")
  logger.Warn("Warning message")
  logger.Error("Error occurred")

  // Log with context
  ctx := xlog.IntoContext(context.Background(), logger)
  xlog.Infof(ctx, "Hello from context: %s", "world")
}
```

Notes:

- The logger prefixes messages with the process ID for easier identification.
- Built using `xlog/rlog` and Logger from the standard library `log` package.
- Log levels can be dynamically changed at runtime.
- Internal log.Logger flags can be customized using `SetFlags(debugFlag, stdFlag int)` on `xlog.Logger`. Offers different flags for when the log level is set to debug or not.

<br>

### xlog/rlog

Package rlog provides a small buffered writer with rotation for logging.

#### Features

- **`Writer`**  
  Provides buffered, size-based log rotation with optional age-based flushing for long running services that want durable logs.

#### Quick example

```go
package main

import (
  "log"
  "time"

  "github.com/Data-Corruption/stdx/xlog/rlog"
)

func main() {
  // Create a new rlog.Writer with rotation and buffering
  w, err := rlog.NewWriter(rlog.Config{
    DirPath:     "./logs",        // required and created if missing
    MaxFileSize: 512 << 20,       // defaults to 256 MB
    MaxBufSize:  8 * 1024,        // defaults to 4 KB
    MaxBufAge:   5 * time.Second, // defaults to 15s (negative to disable)
  })
  if err != nil {
    log.Fatalf("rlog: %v", err)
  }
  defer w.Close()

  // plain io.Writer usage
  log.SetOutput(w)
  log.Println("hello, rotating world")
}
```

Notes & Limitations:

- Rotation renames the active file to a timestamped `<ts>.log` and re-creates `latest.log` atomically. A lightweight file-lock prevents concurrent rotations across processes.
- Only a single rlog.Writer should be used per directory per process; multiple processes may safely share the same directory.

<br>

### xnet

Package `xnet` provides functions for networking.

#### Features

- **`Wait(ctx context.Context, timeout time.Duration, probes ...string error`**  
  Blocks until "the network is probably usable" or ctx/timeout expires.

#### Quick example

```go
package main

import "github.com/Data-Corruption/stdx/xnet"

func main() {
  // wait up to 30s; succeed on any probe
  _ = xnet.Wait(context.Background(), 30*time.Second)

  // OR, if you want to target something closer to your needs:
  // _ = xnet.Wait(ctx, 30*time.Second, "tcp:8.8.8.8:53", "dns:yourdomain.tld")
}
```

<br>

### xterm/prompt

Package `prompt` provides functions for asking interactive questions in the terminal.

#### Features

- **`Int(prompt string) (int, error)`**  
  Re-prompts until the user enters any signed integer.

- **`Uint(prompt string) (uint, error)`**  
  Re-prompts until the user enters a non-negative integer.

- **`String(prompt string) (string, error)`**  
  Reads one line of text (empty string allowed).

- **`YesNo(prompt string) (bool, error)`**  
  Asks a *yes / no* question; returns `true` for “yes”.

#### Quick example

```go
package main

import (
  "fmt"
  "log"

  "github.com/Data-Corruption/stdx/xterm/prompt"
)

func main() {
  age, err := prompt.Int("Enter your age")
  if err != nil {
    log.Fatalf("prompt failed: %v", err)
  }
  fmt.Printf("You are %d years old.\n", age)
}
```
