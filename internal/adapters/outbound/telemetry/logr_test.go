package telemetry_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"

	"github.com/go-logr/logr"

	"github.com/claudioed/facility-layout/internal/adapters/outbound/telemetry"
)

// The point of the bridge is that the SDK's diagnostics come out as JSON
// like every other log line, instead of plain text on stderr.
func TestLogSinkEmitsSDKDiagnosticsAsJSON(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	sink := logr.New(telemetry.NewLogSink(logger))

	sink.WithName("otlp").WithValues("input", "bad endpoint").Error(errors.New("boom"), "parse url")

	var line map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &line); err != nil {
		t.Fatalf("not valid JSON (%q): %v", buf.String(), err)
	}
	if line["level"] != "ERROR" {
		t.Fatalf("expected an error line, got %v", line)
	}
	if line["msg"] != "otlp: parse url" {
		t.Fatalf("expected the logr name to prefix the message, got %v", line["msg"])
	}
	if line["input"] != "bad endpoint" {
		t.Fatalf("WithValues was dropped: %v", line)
	}
	if line["error"] != "boom" {
		t.Fatalf("the error was dropped: %v", line)
	}
}

func TestLogSinkMapsVerbosityToLevel(t *testing.T) {
	tests := []struct {
		name      string
		verbosity int
		want      string
	}{
		{"v0 is operator-facing", 0, "INFO"},
		{"v1 is sdk detail", 1, "DEBUG"},
		{"v8 is sdk detail", 8, "DEBUG"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
			logr.New(telemetry.NewLogSink(logger)).V(tt.verbosity).Info("exporting", "batch", 7)

			var line map[string]any
			if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &line); err != nil {
				t.Fatalf("not valid JSON (%q): %v", buf.String(), err)
			}
			if line["level"] != tt.want {
				t.Fatalf("V(%d) logged at %v, want %v", tt.verbosity, line["level"], tt.want)
			}
			if line["batch"] != float64(7) {
				t.Fatalf("key/values were dropped: %v", line)
			}
		})
	}
}

// SDK detail must not reach the log when the configured level excludes it.
func TestLogSinkRespectsTheConfiguredLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	logr.New(telemetry.NewLogSink(logger)).V(4).Info("chatty")

	if buf.Len() != 0 {
		t.Fatalf("expected debug-level SDK detail to be filtered out, got %q", buf.String())
	}
}

func TestNewLogSinkFallsBackToTheDefaultLogger(t *testing.T) {
	if telemetry.NewLogSink(nil) == nil {
		t.Fatal("expected a usable sink for a nil logger")
	}
}
