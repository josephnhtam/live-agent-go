package helper

import (
	"context"
	"log/slog"
)

var noopHandler slog.Handler = noopSlogHandler{}

func NoopLogger() *slog.Logger {
	return slog.New(noopHandler)
}

type noopSlogHandler struct{}

func (noopSlogHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (noopSlogHandler) Handle(context.Context, slog.Record) error { return nil }
func (h noopSlogHandler) WithAttrs([]slog.Attr) slog.Handler     { return h }
func (h noopSlogHandler) WithGroup(string) slog.Handler           { return h }
