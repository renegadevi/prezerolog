# Pre-zerolog (prezerolog)

[![Go Reference](https://pkg.go.dev/badge/github.com/renegadevi/prezerolog.svg)](https://pkg.go.dev/github.com/renegadevi/prezerolog)
[![License](https://img.shields.io/github/license/renegadevi/prezerolog)](https://github.com/renegadevi/prezerolog/blob/main/LICENSE)
[![Release](https://img.shields.io/github/v/release/renegadevi/prezerolog)](https://github.com/renegadevi/prezerolog/releases)

`InitLogging()` is the simple zerolog setup you would typically write by hand.

One call gives you rotated JSON files, a console writer with its own level, stable field names, UTC timestamps, the caller of your log line, and optional sampling. The logging is simple: `Info`, `Warn`, `Error`, `Debug`, and `Trace`

## Features

- JSON logs in `LOG_DIR`, rotated by size, with a backup count and gzip. Each process start gets a new `service_YYYY-MM-DDTHH-MM-SS_00.log`.
- A console destination that is independent of the file. Text modes go to stderr. `json` mode writes the same JSON to stdout for collectors.
- Separate file and console levels. A debug line can be stored in the file while the console stays at info.
- Four console shapes: `minimal`, `full` (default), `extended`, and `json`.
- Every line carries `time`, `level`, `msg`, `service`, `env`, and `caller`. `time` is UTC `RFC3339Nano`.
- `caller` is the application frame. Frames inside this package are skipped.
- Arguments can be a string message, an `error` (logged as `err`), a `map[string]any`, an exported struct, or a `fmt.Stringer`.
- `Fatal` exits 1. `FatalCode` exits with the code you pass. When the console is enabled, the fatal line includes a stack.
- `LOG_SAMPLING_N` keeps 1 of every N events (the 1st, then every Nth after that). `1` keeps every event.
- A `.env` file in the working directory is loaded when it is present. Values already set in the environment stay the same.

## Install

```bash
go get github.com/renegadevi/prezerolog
```

## Quick start

```go
package main

import (
	"errors"

	log "github.com/renegadevi/prezerolog"
)

func main() {
	// Loads .env when present, then opens the file and console writers.
	log.InitLogging()

	log.Info("Logging initialized")
	log.Warn("This is a warning message")
	log.Error("This is an error message", errors.New("example error"))
	log.Debug("This is a debug message", map[string]any{"debug": true})
	log.Trace("This is a trace message", map[string]any{"key": "value"})
}
```

See [examples/basic](examples/basic/main.go) and [examples/json](examples/json/main.go).

## Screenshots

![screenshot-env](https://raw.githubusercontent.com/renegadevi/prezerolog/main/.github/screenshot-env.png)
![screenshot-json](https://raw.githubusercontent.com/renegadevi/prezerolog/main/.github/screenshot-json-stdout.png)
![screenshot-file](https://raw.githubusercontent.com/renegadevi/prezerolog/main/.github/screenshot-file.png)

## Log files and rotation

File logs are JSON, written to `LOG_DIR` (default `logs/`). The base name is the sanitized service name plus the process start time. The suffix starts at `_00` and increments when the file rotates.

**Fresh start:**

```
logs/
└─ my-service_2025-08-10T15-10-22_00.log
```

**After a few rotations:**

```
logs/
├─ my-service_2025-08-10T15-10-22_00.log
├─ my-service_2025-08-10T15-10-22_01.log
└─ my-service_2025-08-10T15-10-22_02.log
```

- `LOG_ROTATE_MAX_SIZE` is the size in MB before a new file starts (default `100`).
- `LOG_ROTATE_MAX_BACKUPS` is how many old files to keep (default `7`).
- Rotated files are gzipped.

The service name comes from `LOG_NAME`. When that is unset, it comes from the main module path, then the working directory, then the binary name. Spaces and slashes become hyphens in the filename.

**Example JSON line:**

```json
{
  "time": "2025-08-10T13:45:15.123456789Z",
  "level": "info",
  "msg": "login",
  "service": "my-service",
  "env": "development",
  "caller": "auth.go:88",
  "user": "ada",
  "role": "admin"
}
```

An `error` argument is added as `err`. The last string argument is the message. Exported struct fields are added under their Go names.

## Console output

`LOG_CONSOLE_OUTPUT` selects the stderr (or stdout) shape:

- `minimal` — time, level, and message.
- `full` (default) — time, level, caller, message, and any fields you passed. `service` and `env` are in the JSON file.
- `extended` — the same parts as `full`, and `service`, `env`, and every other field are shown.
- `json` — one JSON object per line on stdout, the same shape as the file.

Text modes write ANSI color. `json` mode writes plain JSON.

When `LOG_FILE` and `LOG_CONSOLE` are both false, JSON is written to stdout at info level so a process with no destination still emits logs.

## Fatal

```go
log.Fatal("unrecoverable error", err)       // exits 1
log.FatalCode(2, "unrecoverable error", err) // exits 2
```

The line is logged at `fatal` before the process exits. When console output is enabled, the line includes a `stack`.

## Environment variables

```ini
LOG_DIR=<folder name>                                      # default: logs
LOG_NAME=<service name>                                    # default: module, directory, or binary
LOG_ENV=production|development                             # default: production, or APP_ENV when set
LOG_FILE=true|false                                        # default: true
LOG_CONSOLE=true|false                                     # default: true
LOG_CONSOLE_LEVEL=trace|debug|info|warn|error|fatal|panic  # default: info
LOG_FILE_LEVEL=trace|debug|info|warn|error|fatal|panic     # default: info
LOG_CONSOLE_OUTPUT=minimal|full|extended|json              # default: full
LOG_SAMPLING_N=1                                           # default: 1, keep every event
LOG_ROTATE_MAX_SIZE=100                                    # megabytes
LOG_ROTATE_MAX_BACKUPS=7
```

`warning` is accepted as `warn`. An unknown level falls back to info. Boolean values accept `1`, `t`, `true`, `y`, `yes`, `on`, and the matching false words.

`trace` is a valid level. Zerolog's process-wide floor starts at debug, so a trace line is emitted when that floor is trace or lower.

## Integration

Call `InitLogging` once from `main`, before other goroutines log. It sets zerolog's process-wide field names (`time`, `level`, `msg`, `caller`, `err`) for the whole process.

## License

MIT 2026 — Philip Andersen
