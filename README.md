# stdx

[![Go Reference](https://pkg.go.dev/badge/github.com/Data-Corruption/stdx.svg)](https://pkg.go.dev/github.com/Data-Corruption/stdx)

Production-hardened extensions for Go's standard library with zero dependencies

```bash
go get github.com/Data-Corruption/stdx
```

## xhttp

Package xhttp provides extensions to the standard net/http package for production-ready HTTP server functionality.

### Features

- **`Server`**  
  A wrapper around `http.Server` that provides:
  - Signal-based graceful shutdown
  - Lifecycle hooks for actions after listening and before shutdown
  - Sensible defaults for server configuration
- **`Err`**  
  A custom error type for HTTP handlers that separates internal errors from client-safe messages.
- **`Error(ctx context.Context, w http.ResponseWriter, err error)`**  
  A function to handle errors in HTTP handlers, logging them and sending appropriate HTTP responses. A drop-in replacement for `http.Error` that works with the `xlog` logger in the context if present.

### Quick example

```golang
// Visit http://localhost:8080 to see a success response or, ~50% of the time,
// an internal error handled by xhttp.Error. Press Ctrl‑C to trigger a graceful
// shutdown.
package main

import (
  "context"
  "errors"
  "fmt"
  "log"
  "math/rand"
  "net/http"
  "time"

  "github.com/Data-Corruption/stdx/xhttp"
)

func main() {
  // Seed rand for the demo error path.
  rand.Seed(time.Now().UnixNano())

  // Build the server with sensible defaults for localhost testing.
  srv, err := xhttp.NewServer(&xhttp.ServerConfig{
    Addr:    ":8080",
    Handler: rootHandler(),
    // AfterListen runs once the listener is up; useful for readiness probes.
    AfterListen: func() {
      log.Println("server is ready and listening on", srv.Addr())
    },
    // Shutdown callback.
    OnShutdown: func() {
      log.Println("shutting down, cleaning up resources …")
    },
    // See xhttp.ServerConfig for all options and defaults.
  })
  if err != nil {
    log.Fatalf("failed to create server: %v", err)
  }

  // Start serving (blocks until exit signal or error).
  log.Fatal(srv.Listen())
}

// rootHandler returns an http.Handler that uses xhttp.Error for responses.
func rootHandler() http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    if err := riskyOperation(); err != nil {
      // Logs then sends either the safe message or "Internal Server Error" if
      // not an xhttp.Err. Will also use xlog Logger if present in context
      xhttp.Error(r.Context(), w, err)
      return
    }

    fmt.Printf(w, "Hello, stdx!")
  })
}

// riskyOperation simulates work that can fail.
func riskyOperation() error {
  if rand.Intn(2) == 0 {
    // Wrap the low‑level error with an xhttp.Err so callers get both
    // the public message and the root cause.
    return &xhttp.Err{
      Code: http.StatusInternalServerError,
      Msg:  "Something went wrong, please try again later", // safe for clients
      Err:  errors.New("simulated failure"),                // internal detail
    }
  }
  // Pretend work succeeds.
  time.Sleep(100 * time.Millisecond)
  return nil
}
```

## xlog

Package xlog provides a leveled, concurrent-safe logger with buffered rotation for logging.

### Features

- **`Logger`**  
  A leveled logger that supports dynamic log level changes, custom formatting, and safe shutdown. Internally uses Writer from `xlog/rlog`.

### Quick example

```golang
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

## xlog/rlog

Package rlog provides a small buffered writer with rotation for logging.

### Features

- **`Writer`**  
  Provides buffered, size-based log rotation with optional age-based flushing for long running services that want durable logs.

### Quick example

```golang
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

## xterm/prompt

Package `prompt` provides functions for asking interactive questions in the terminal.

### Features

- **`Int(prompt string) (int, error)`**  
  Re-prompts until the user enters any signed integer.

- **`Uint(prompt string) (uint, error)`**  
  Re-prompts until the user enters a non-negative integer.

- **`String(prompt string) (string, error)`**  
  Reads one line of text (empty string allowed).

- **`YesNo(prompt string) (bool, error)`**  
  Asks a *yes / no* question; returns `true` for “yes”.

### Quick example

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
