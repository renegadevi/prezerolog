# Changelog

## v.1.0.2 - 2026-10-18
- Revert README because it didn't read simple enough.

## v.1.0.1 - 2026-10-18
- Added tests for the JSON line, levels, sampling, caller, and the fatal exit status.
- Rewrote the README around what InitLogging sets up.

## v1.0.0 - 2026-10-18
- Updated dependencies  (zerolog v1.35.1, go-colorable v0.1.14, golang.org/x/sys v0.29.0)
- FatalCode and FatalError.Code now exit with the requested status. zerolog's Fatal hook was exiting 1 before that status ran. Fatal with no code still exits 1.

## v0.4.0 - 2025-08-12
- Added JSON to stdout using LOG_CONSOLE_OUTPUT=json
- Added LOG_FILE=false to disable file logging
- Added json example (`examples/json/main.go`)

## v0.3.0 - 2025-08-11
- Updated environmental variables to be specific to LOG_
- Updated example demo code
- Updated fatal code
- Updated Screenshots with env/json
- Updated README and CHANGELOG

## v0.2.0 — 2025-08-10
- Updated exit codes/fatal
- Working caller code

## v0.1.1 — 2025-08-10
- Updated caller code
- Updated README with screenshots

## v0.1.0 — 2025-08-10
- Initial preview release of `prezerolog`
- JSON logs to rotated files (lumberjack)
- Colored console output in development/DEBUG
- Normalized keys: `time, level, msg, service, env, trace_id, span_id, request_id, caller, err`
- RFC3339Nano timestamps (UTC)
- Context-aware correlation IDs
- Fatal with stack (dev) and exit code
- Package-level helpers (`prezerolog.Info`, `prezerolog.InfoCtx`, etc.)
