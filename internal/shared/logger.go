package shared

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strconv"
	"time"
)

type Logger struct {
	*slog.Logger
}

func NewLogger(env string) *Logger {
	var h slog.Handler

	opts := &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: env == "development",
	}

	if env == "development" {
		h = NewPrettyHandler(os.Stdout, opts)
	} else {
		h = slog.NewJSONHandler(os.Stdout, opts)
	}

	l := &Logger{slog.New(h)}
	slog.SetDefault(l.Logger)
	return l
}

func (l *Logger) ErrorCtx(ctx context.Context, msg string, attrs ...any) {
	l.LogAttrs(ctx, slog.LevelError, msg, toAttrs(attrs)...)
}

func (l *Logger) InfoCtx(ctx context.Context, msg string, attrs ...any) {
	l.LogAttrs(ctx, slog.LevelInfo, msg, toAttrs(attrs)...)
}

func (l *Logger) DebugCtx(ctx context.Context, msg string, attrs ...any) {
	l.LogAttrs(ctx, slog.LevelDebug, msg, toAttrs(attrs)...)
}

func (l *Logger) WarnCtx(ctx context.Context, msg string, attrs ...any) {
	l.LogAttrs(ctx, slog.LevelWarn, msg, toAttrs(attrs)...)
}

func toAttrs(attrs []any) []slog.Attr {
	out := make([]slog.Attr, 0, len(attrs)/2)
	for i := 0; i+1 < len(attrs); i += 2 {
		key, ok := attrs[i].(string)
		if !ok {
			continue
		}
		out = append(out, slog.Any(key, attrs[i+1]))
	}
	return out
}

func SourceAttr() slog.Attr {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		return slog.Attr{}
	}
	return slog.String("source", file+":"+strconv.Itoa(line))
}

func DurationAttr(d time.Duration) slog.Attr {
	return slog.String("duration", d.String())
}

func PrintBanner(port, env string) {
	url := fmt.Sprintf("http://localhost:%s", port)

	fmt.Println()
	fmt.Printf("%s╭─── Azzet ───────────────────────────────────────────────╮%s\n", cyan, reset)
	fmt.Printf("%s│                                                         │%s\n", cyan, reset)
	fmt.Printf("%s│   %sAccounting, Tax & Finance Platform%s                    │%s\n", cyan, bold+white, reset, cyan)
	fmt.Printf("%s│                                                         │%s\n", cyan, reset)
	fmt.Printf("%s│   %s●%s API      %s→%s %s%-41s%s│%s\n", cyan, green, reset, bold, reset, white, url+"/api/v1", cyan, reset)
	fmt.Printf("%s│   %s●%s Swagger  %s→%s %s%-41s%s│%s\n", cyan, green, reset, bold, reset, white, url+"/swagger/index.html", cyan, reset)
	fmt.Printf("%s│   %s●%s Env      %s→%s %s%-41s%s│%s\n", cyan, green, reset, bold, reset, yellow, env, cyan, reset)
	fmt.Printf("%s│                                                         │%s\n", cyan, reset)
	fmt.Printf("%s╰─────────────────────────────────────────────────────────╯%s\n", cyan, reset)
	fmt.Println()
}
