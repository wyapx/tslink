package core

import (
	"context"
	"os"
	"strings"
	"time"

	"log/slog"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-colorable"
)

type uiLogHandler struct {
	level slog.Level
}

func (h *uiLogHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *uiLogHandler) Handle(_ context.Context, r slog.Record) error {
	if Events != nil {
		select {
		case Events <- &LogEvent{Message: r.Message}:
		default:
			// drop log if channel is full to avoid blocking
		}
	}
	return nil
}

func (h *uiLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *uiLogHandler) WithGroup(name string) slog.Handler {
	return h
}

type multiHandler struct {
	handlers []slog.Handler
}

func (m *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range m.handlers {
		if h.Enabled(ctx, r.Level) {
			h.Handle(ctx, r)
		}
	}
	return nil
}

func (m *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		newHandlers[i] = h.WithAttrs(attrs)
	}
	return &multiHandler{handlers: newHandlers}
}

func (m *multiHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		newHandlers[i] = h.WithGroup(name)
	}
	return &multiHandler{handlers: newHandlers}
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func NewLogger(level string, useJsonFormat bool) *slog.Logger {
	w := os.Stdout
	var mainHandler slog.Handler
	lvl := parseLevel(level)
	if !useJsonFormat {
		mainHandler = tint.NewHandler(colorable.NewColorable(w), &tint.Options{
			Level:      lvl,
			TimeFormat: time.DateTime,
			//NoColor:    !isatty.IsTerminal(w.Fd()),
		})
	} else {
		mainHandler = slog.NewJSONHandler(w, &slog.HandlerOptions{
			Level: lvl,
		})
	}

	uiH := &uiLogHandler{level: lvl}
	logger := slog.New(&multiHandler{handlers: []slog.Handler{mainHandler, uiH}})

	slog.SetDefault(logger)
	return logger
}
