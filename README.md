# Pre-zerolog (prezerolog)

[![Go Reference](https://pkg.go.dev/badge/github.com/renegadevi/prezerolog.svg)](https://pkg.go.dev/github.com/renegadevi/prezerolog)
[![License](https://img.shields.io/github/license/renegadevi/prezerolog)](https://github.com/renegadevi/prezerolog/blob/main/LICENSE)
[![Release](https://img.shields.io/github/v/release/renegadevi/prezerolog)](https://github.com/renegadevi/prezerolog/releases)


A logging wrapper around [zerolog](https://github.com/rs/zerolog) with:
- JSON logs to rotated files (or stdout, if you want to ingest them)
- Colored console output
- Separate levels for the log file and the console
- Console layout: `minimal`, `full`, `extended`, or `json`
- Same field names on every line: `time`, `level`, `msg`, `service`, `env`, `caller`, `err`
- RFC3339Nano timestamps (UTC)
- Pass a string, an error, a map, or a struct
- `Fatal` / `FatalCode` with your exit code (default 1), and a stack when the console is on

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
	// Initialize logging (loads .env if present, sets up rotation & console)
	log.InitLogging()

	// Logging examples (check the examples/ folder for more)
	log.Info("Logging initialized")
	log.Warn("This is a warning message")
	log.Error("This is an error message", errors.New("example error"))
	log.Debug("This is a debug message", map[string]any{"debug": true})
	log.Trace("This is a trace message", map[string]any{"key": "value"})

}

```


## Screenshots

![screenshot-env](https://raw.githubusercontent.com/renegadevi/prezerolog/main/.github/screenshot-env.png)
![screenshot-json](https://raw.githubusercontent.com/renegadevi/prezerolog/main/.github/screenshot-json-stdout.png)
![screenshot-file](https://raw.githubusercontent.com/renegadevi/prezerolog/main/.github/screenshot-file.png)


## Log files & rotation

By default, logs are written to `LOG_DIR` (default: `logs/`) as JSON.
Each run uses a date-based base name and a zero-padded index that increments on rotation.

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

Where:
- `my-service_YYYY-MM-DDTHH-MM-SS` = base name for the current process start
- `_00.log` = the first file for this run; subsequent files increment to `_01.log`, `_02.log`, …

Rotation is controlled by environment variables:
- `LOG_ROTATE_MAX_SIZE` (MB, default `100`)
- `LOG_ROTATE_MAX_BACKUPS` (default `7`, older files are removed beyond this count)
- Files are compressed (`.gz`) automatically when rotated.

The service name is `LOG_NAME`. If that is empty, it uses the module name, then the current directory, then the binary name.

**Example JSON line:**
```json
{
  "time":"2025-08-10T13:45:15.123456789Z",
  "level":"info",
  "msg":"login",
  "service":"my-service",
  "env":"development",
  "caller":"auth.go:88",
  "user":"john",
  "role":"admin"
}
```

## Keys
All records include:

`time`, `level`, `msg`, `service`, `env`, `caller`

`caller` is the file and line that logged, not this package.
Errors passed as `error` args are logged under `err`.
A map is copied in as fields. A struct adds its exported fields under their Go names. The last string is the message.

## Console output
`LOG_CONSOLE_OUTPUT` (default `full`):
- `minimal`: time, level, message
- `full`: time, level, caller, message, and your fields. `service` and `env` stay in the file
- `extended`: same as `full`, and also shows `service` and `env`
- `json`: the same JSON as the file, written to stdout

Text modes are colored and go to stderr. If both `LOG_FILE` and `LOG_CONSOLE` are false, JSON goes to stdout.

## Environment variables
```ini
LOG_DIR=<folder name>                         # (default: "logs")
LOG_NAME=<service name>                       # (default: module name, else current dir)
LOG_ENV=production|development                # (default: production, or APP_ENV if set)
LOG_FILE=true|false                           # (default: true)
LOG_CONSOLE=true|false                        # (default: true)
LOG_CONSOLE_LEVEL=trace|debug|info|warn|error|fatal|panic  # (default: info)
LOG_FILE_LEVEL=trace|debug|info|warn|error|fatal|panic     # (default: info)
LOG_CONSOLE_OUTPUT=minimal|full|extended|json # (default: full)
LOG_SAMPLING_N=1                              # (default: 1, no sampling)
LOG_ROTATE_MAX_SIZE=100                       # (default: 100)
LOG_ROTATE_MAX_BACKUPS=7                      # (default: 7)
```

`warning` is accepted as `warn`. An unknown level falls back to info.
`LOG_SAMPLING_N=2` keeps the 1st event, then every 2nd after that.
A `.env` file in the working directory is loaded when it is present. Variables already set in the environment are left as they are.

## Fatal
```go
log.Fatal("unrecoverable error", err)        // exits 1
log.FatalCode(2, "unrecoverable error", err) // exits 2
```

The line is logged at `fatal`. A stack is included when the console is on.


## License
MIT 2025-2026 - Philip Andersen
