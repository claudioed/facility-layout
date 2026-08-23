package telemetry

import (
	"context"
	"log/slog"

	"github.com/go-logr/logr"
)

// The OpenTelemetry SDK reports its own diagnostics (a malformed endpoint,
// a dropped batch) through a logr.Logger that writes plain text to stderr
// by default. That would be the one unstructured stream in an otherwise
// all-JSON log, so it is bridged onto slog here.
//
// logr's verbosity is inverted relative to slog's levels: V(0) is the most
// important. Anything above V(0) is SDK-internal detail and lands at debug.
type slogSink struct {
	logger *slog.Logger
	name   string
	attrs  []any
}

// NewLogSink returns a logr.LogSink that forwards to logger. Install it
// with otel.SetLogger(logr.New(NewLogSink(logger))).
func NewLogSink(logger *slog.Logger) logr.LogSink {
	if logger == nil {
		logger = slog.Default()
	}
	return &slogSink{logger: logger}
}

func (s *slogSink) Init(logr.RuntimeInfo) {}

func (s *slogSink) Enabled(level int) bool {
	return s.logger.Enabled(context.Background(), verbosityToLevel(level))
}

func (s *slogSink) Info(level int, msg string, keysAndValues ...any) {
	s.logger.Log(context.Background(), verbosityToLevel(level), s.prefixed(msg), s.merge(keysAndValues)...)
}

func (s *slogSink) Error(err error, msg string, keysAndValues ...any) {
	s.logger.Error(s.prefixed(msg), append(s.merge(keysAndValues), "error", err)...)
}

func (s *slogSink) WithValues(keysAndValues ...any) logr.LogSink {
	return &slogSink{logger: s.logger, name: s.name, attrs: s.merge(keysAndValues)}
}

func (s *slogSink) WithName(name string) logr.LogSink {
	joined := name
	if s.name != "" {
		joined = s.name + "/" + name
	}
	return &slogSink{logger: s.logger, name: joined, attrs: s.attrs}
}

// prefixed keeps the logr logger's name visible in the flat slog message,
// which is how the SDK says which of its components spoke.
func (s *slogSink) prefixed(msg string) string {
	if s.name == "" {
		return msg
	}
	return s.name + ": " + msg
}

func (s *slogSink) merge(keysAndValues []any) []any {
	if len(s.attrs) == 0 {
		return keysAndValues
	}
	merged := make([]any, 0, len(s.attrs)+len(keysAndValues))
	merged = append(merged, s.attrs...)
	merged = append(merged, keysAndValues...)
	return merged
}

// verbosityToLevel maps a logr verbosity onto a slog level: V(0) is the
// SDK saying something an operator should see, anything louder is detail.
func verbosityToLevel(verbosity int) slog.Level {
	if verbosity <= 0 {
		return slog.LevelInfo
	}
	return slog.LevelDebug
}
