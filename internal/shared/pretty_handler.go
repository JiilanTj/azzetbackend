package shared

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"sync"
)

const (
	reset = "\033[0m"
	bold  = "\033[1m"

	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	blue   = "\033[34m"
	cyan   = "\033[36m"
	white  = "\033[37m"
	gray   = "\033[90m"
)

type PrettyHandler struct {
	opts  slog.HandlerOptions
	attrs []slog.Attr
	group string
	mu    sync.Mutex
	w     io.Writer
}

func NewPrettyHandler(w io.Writer, opts *slog.HandlerOptions) *PrettyHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	if opts.Level == nil {
		opts.Level = slog.LevelInfo
	}
	return &PrettyHandler{w: w, opts: *opts}
}

func (h *PrettyHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

func (h *PrettyHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	ts := r.Time.Format("15:04:05.000")
	fmt.Fprintf(h.w, "%s%s%s ", gray, ts, reset)

	lvl, lvlClr := levelColor(r.Level)
	fmt.Fprintf(h.w, "%s%s%s ", lvlClr, lvl, reset)

	fmt.Fprintf(h.w, "%s%s%s", bold, r.Message, reset)

	attrs := h.attrs
	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, a)
		return true
	})

	attrs = flatten(attrs)

	for _, a := range attrs {
		val := a.Value.String()
		if val == "" || val == "<nil>" {
			continue
		}
		fmt.Fprintf(h.w, "  %s●%s %s%s%s=%s%s%s", gray, reset, cyan, a.Key, reset, white, val, reset)
	}

	if h.opts.AddSource && r.PC != 0 {
		frames := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := frames.Next()
		shortFile := shortenPath(f.File)
		fmt.Fprintf(h.w, "  %s(%s:%d)%s", gray, shortFile, f.Line, reset)
	}

	fmt.Fprintln(h.w)
	return nil
}

func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &PrettyHandler{
		w:     h.w,
		opts:  h.opts,
		group: h.group,
		attrs: append(h.attrs, attrs...),
	}
}

func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	return &PrettyHandler{
		w:     h.w,
		opts:  h.opts,
		group: name,
		attrs: h.attrs,
	}
}

func levelColor(l slog.Level) (string, string) {
	switch {
	case l < slog.LevelInfo:
		return "DEBUG", cyan
	case l < slog.LevelWarn:
		return "INFO", green
	case l < slog.LevelError:
		return "WARN", yellow
	default:
		return "ERROR", red
	}
}

func flatten(attrs []slog.Attr) []slog.Attr {
	var out []slog.Attr
	for _, a := range attrs {
		if a.Value.Kind() == slog.KindGroup {
			out = append(out, flatten(a.Value.Group())...)
		} else {
			out = append(out, a)
		}
	}
	return out
}

func shortenPath(path string) string {
	start := 0
	for i := 0; i < 2; i++ {
		idx := indexAt(path, '/', start)
		if idx == -1 {
			break
		}
		start = idx + 1
	}
	if start >= len(path) {
		return path
	}
	return path[start:]
}

func indexAt(s string, c byte, start int) int {
	for i := start; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}
