package prezerolog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestJSONLineShape(t *testing.T) {
	path := setupFileLogger(t, nil)
	Info("hello")

	recs := readRecords(t, path)
	if len(recs) != 1 {
		t.Fatalf("records = %d, want 1", len(recs))
	}
	rec := recs[0]
	assertKeys(t, rec, "time", "level", "msg", "service", "env", "caller")
	if rec["level"] != "info" || rec["msg"] != "hello" || rec["service"] != "svc" || rec["env"] != "test" {
		t.Fatalf("record = %#v", rec)
	}
	assertUTCTimestamp(t, rec["time"])
	caller, _ := rec["caller"].(string)
	if caller == "" || strings.Contains(caller, "prezerolog.go") {
		t.Fatalf("caller = %q, want an application frame", caller)
	}
}

func TestLogArgs(t *testing.T) {
	path := setupFileLogger(t, nil)

	Info("first", "second")
	Error("failed", errors.New("boom"), map[string]any{"user": "ada"})
	Info("payload", samplePayload{User: "ada", Count: 2, hidden: "nope"})
	Info(sampleLabel("only-stringer"))
	Info("has-message", sampleLabel("extra"))

	recs := readRecords(t, path)
	if len(recs) != 5 {
		t.Fatalf("records = %d, want 5: %#v", len(recs), recs)
	}

	if recs[0]["msg"] != "second" {
		t.Fatalf("last string = %#v", recs[0])
	}
	if recs[1]["msg"] != "failed" || recs[1]["err"] != "boom" || recs[1]["user"] != "ada" {
		t.Fatalf("error+map = %#v", recs[1])
	}
	if recs[2]["msg"] != "payload" || recs[2]["User"] != "ada" || recs[2]["Count"] != float64(2) {
		t.Fatalf("struct = %#v", recs[2])
	}
	if _, ok := recs[2]["hidden"]; ok {
		t.Fatalf("unexported field leaked: %#v", recs[2])
	}
	if recs[3]["msg"] != "only-stringer" {
		t.Fatalf("stringer message = %#v", recs[3])
	}
	if _, ok := recs[3]["value"]; ok {
		t.Fatalf("stringer used as message also set value: %#v", recs[3])
	}
	if recs[4]["msg"] != "has-message" || recs[4]["value"] != "extra" {
		t.Fatalf("stringer field = %#v", recs[4])
	}
}

func TestContextIDs(t *testing.T) {
	path := setupFileLogger(t, nil)

	ctx := context.WithValue(context.Background(), CtxRequestID, "req-1")
	ctx = context.WithValue(ctx, CtxTraceID, "tr-1")
	ctx = context.WithValue(ctx, CtxSpanID, "sp-1")
	InfoCtx(ctx, "with-ids")

	empty := context.WithValue(context.Background(), CtxRequestID, "")
	InfoCtx(empty, "empty-id")
	Info("plain")

	recs := readRecords(t, path)
	if len(recs) != 3 {
		t.Fatalf("records = %d, want 3", len(recs))
	}
	if recs[0]["request_id"] != "req-1" || recs[0]["trace_id"] != "tr-1" || recs[0]["span_id"] != "sp-1" {
		t.Fatalf("context ids = %#v", recs[0])
	}
	for _, key := range []string{"request_id", "trace_id", "span_id"} {
		if _, ok := recs[1][key]; ok {
			t.Fatalf("empty id recorded on %s: %#v", key, recs[1])
		}
		if _, ok := recs[2][key]; ok {
			t.Fatalf("plain log recorded %s: %#v", key, recs[2])
		}
	}
}

func TestFileLevelFilter(t *testing.T) {
	path := setupFileLogger(t, map[string]string{"LOG_FILE_LEVEL": "error"})
	Info("dropped")
	Error("kept", errors.New("x"))

	recs := readRecords(t, path)
	if len(recs) != 1 || recs[0]["msg"] != "kept" || recs[0]["level"] != "error" {
		t.Fatalf("records = %#v", recs)
	}
}

func TestUnknownLevelFallsBackToInfo(t *testing.T) {
	path := setupFileLogger(t, map[string]string{"LOG_FILE_LEVEL": "nope"})
	Debug("dropped")
	Info("kept")

	recs := readRecords(t, path)
	if len(recs) != 1 || recs[0]["msg"] != "kept" {
		t.Fatalf("records = %#v", recs)
	}
}

func TestSamplingKeepsEveryNthStartingAtFirst(t *testing.T) {
	path := setupFileLogger(t, map[string]string{"LOG_SAMPLING_N": "2"})
	for _, msg := range []string{"s1", "s2", "s3", "s4"} {
		Info(msg)
	}

	recs := readRecords(t, path)
	got := make([]string, 0, len(recs))
	for _, rec := range recs {
		got = append(got, fmt.Sprint(rec["msg"]))
	}
	if strings.Join(got, ",") != "s1,s3" {
		t.Fatalf("sampled = %v, want [s1 s3]", got)
	}
}

func TestMultiLevelWriterFilters(t *testing.T) {
	var low, high bytes.Buffer
	w := multiLevelWriter{dests: []levelDest{
		{w: &low, min: zerolog.InfoLevel},
		{w: &high, min: zerolog.ErrorLevel},
	}}
	write := func(level zerolog.Level, p string) {
		t.Helper()
		n, err := w.WriteLevel(level, []byte(p))
		if err != nil || n != len(p) {
			t.Fatalf("WriteLevel(%s) n=%d err=%v", level, n, err)
		}
	}
	write(zerolog.DebugLevel, "debug")
	write(zerolog.InfoLevel, "info")
	write(zerolog.WarnLevel, "warn")
	write(zerolog.ErrorLevel, "error")

	if low.String() != "infowarnerror" {
		t.Fatalf("low = %q", low.String())
	}
	if high.String() != "error" {
		t.Fatalf("high = %q", high.String())
	}
}

func TestParseLevel(t *testing.T) {
	cases := []struct {
		in   string
		want zerolog.Level
	}{
		{"warning", zerolog.WarnLevel},
		{"warn", zerolog.WarnLevel},
		{"error", zerolog.ErrorLevel},
		{"fatal", zerolog.FatalLevel},
		{"panic", zerolog.PanicLevel},
		{"trace", zerolog.TraceLevel},
		{"debug", zerolog.DebugLevel},
		{"info", zerolog.InfoLevel},
		{"", zerolog.InfoLevel},
		{"nope", zerolog.NoLevel},
	}
	for _, tc := range cases {
		if got := parseLevel(tc.in); got != tc.want {
			t.Errorf("parseLevel(%q) = %s, want %s", tc.in, got, tc.want)
		}
	}
}

func TestCallerIsApplicationFrame(t *testing.T) {
	dir := t.TempDir()
	if strings.Contains(filepath.ToSlash(dir), "/prezerolog/") {
		t.Fatal("temp dir is inside this module, so the caller walk would skip the application frame")
	}
	logDir := filepath.Join(dir, "logs")
	mod := `module example.com/callercheck

go 1.24.5

require github.com/renegadevi/prezerolog v0.0.0

replace github.com/renegadevi/prezerolog => ` + filepath.ToSlash(moduleRoot(t)) + "\n"
	mustWriteFile(t, filepath.Join(dir, "go.mod"), mod)
	mustWriteFile(t, filepath.Join(dir, "caller_test.go"), `package callercheck

import (
	"testing"

	log "github.com/renegadevi/prezerolog"
)

func TestFromApp(t *testing.T) {
	log.InitLogging()
	log.Info("from-app")
}
`)

	env := toolEnv(
		"LOG_DIR="+logDir,
		"LOG_NAME=callerapp",
		"LOG_ENV=test",
		"LOG_FILE=true",
		"LOG_CONSOLE=false",
		"LOG_FILE_LEVEL=info",
		"LOG_SAMPLING_N=1",
	)
	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = dir
	tidy.Env = env
	if out, err := tidy.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed: %v\n%s", err, out)
	}

	cmd := exec.Command("go", "test", "-count=1", "-timeout", "120s", ".")
	cmd.Dir = dir
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("external test failed: %v\n%s", err, out)
	}

	recs := readRecords(t, loneLogFile(t, logDir))
	if len(recs) != 1 || recs[0]["msg"] != "from-app" {
		t.Fatalf("records = %#v", recs)
	}
	caller, _ := recs[0]["caller"].(string)
	if !strings.HasPrefix(caller, "caller_test.go:") {
		t.Fatalf("caller = %q, want caller_test.go:line", caller)
	}
}

func TestFatalCode(t *testing.T) {
	if os.Getenv("PREZEROLOG_FATAL_CHILD") == "1" {
		InitLogging()
		FatalCode(3, "boom")
		os.Exit(1)
	}

	dir := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestFatalCode$", "-test.count=1")
	cmd.Env = toolEnv(
		"PREZEROLOG_FATAL_CHILD=1",
		"LOG_DIR="+dir,
		"LOG_NAME=fataltest",
		"LOG_ENV=test",
		"LOG_FILE=true",
		"LOG_CONSOLE=false",
		"LOG_FILE_LEVEL=info",
		"LOG_SAMPLING_N=1",
	)
	out, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected exit, got %v\n%s", err, out)
	}
	if code := exitErr.ExitCode(); code != 3 {
		t.Fatalf("exit = %d, want 3\n%s", code, out)
	}

	recs := readRecords(t, loneLogFile(t, dir))
	if len(recs) != 1 || recs[0]["level"] != "fatal" || recs[0]["msg"] != "boom" {
		t.Fatalf("fatal record = %#v", recs)
	}
}

type samplePayload struct {
	User   string
	Count  int
	hidden string
}

type sampleLabel string

func (s sampleLabel) String() string { return string(s) }

func setupFileLogger(t *testing.T, extra map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	env := map[string]string{
		"LOG_DIR":            dir,
		"LOG_NAME":           "svc",
		"LOG_ENV":            "test",
		"LOG_FILE":           "true",
		"LOG_CONSOLE":        "false",
		"LOG_FILE_LEVEL":     "trace",
		"LOG_CONSOLE_LEVEL":  "info",
		"LOG_CONSOLE_OUTPUT": "full",
		"LOG_SAMPLING_N":     "1",
	}
	for k, v := range extra {
		env[k] = v
	}
	for k, v := range env {
		t.Setenv(k, v)
	}
	InitLogging()
	path := AppLogger.GetCurrentLogFile()
	if path == "" {
		t.Fatal("log file path is empty")
	}
	return path
}

func readRecords(t *testing.T, path string) []map[string]any {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	var recs []map[string]any
	for i, line := range bytes.Split(b, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var rec map[string]any
		if err := json.Unmarshal(line, &rec); err != nil {
			t.Fatalf("line %d: %v\n%s", i+1, err, line)
		}
		recs = append(recs, rec)
	}
	return recs
}

func assertKeys(t *testing.T, rec map[string]any, keys ...string) {
	t.Helper()
	if len(rec) != len(keys) {
		t.Fatalf("keys = %v, want %v", mapKeys(rec), keys)
	}
	for _, k := range keys {
		if _, ok := rec[k]; !ok {
			t.Fatalf("missing %s in %#v", k, rec)
		}
	}
}

func assertUTCTimestamp(t *testing.T, v any) {
	t.Helper()
	s, ok := v.(string)
	if !ok {
		t.Fatalf("time = %#v", v)
	}
	ts, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t.Fatalf("time %q: %v", s, err)
	}
	if ts.Location() != time.UTC || !strings.HasSuffix(s, "Z") {
		t.Fatalf("time %q is not UTC RFC3339Nano", s)
	}
}

func loneLogFile(t *testing.T, dir string) string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, "*.log"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("log files = %v", matches)
	}
	return matches[0]
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return wd
}

func mustWriteFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mapKeys(rec map[string]any) []string {
	keys := make([]string, 0, len(rec))
	for k := range rec {
		keys = append(keys, k)
	}
	return keys
}

func toolEnv(extra ...string) []string {
	keys := []string{
		"PATH", "HOME", "USER", "TMPDIR",
		"GOCACHE", "GOMODCACHE", "GOPATH", "GOROOT",
		"GOTOOLCHAIN", "GOPROXY", "GOSUMDB", "GOROOT_FINAL",
	}
	env := make([]string, 0, len(keys)+len(extra)+1)
	for _, key := range keys {
		if v, ok := os.LookupEnv(key); ok {
			env = append(env, key+"="+v)
		}
	}
	env = append(env, "GOWORK=off")
	env = append(env, extra...)
	return env
}
