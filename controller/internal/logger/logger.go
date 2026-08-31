package logger

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"strings"
	"sync"
)

type Level int

const (
	LevelTrace Level = iota
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelTrace:
		return "TRACE"
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "INFO"
	}
}

func (l Level) SlogLevel() slog.Level {
	switch l {
	case LevelTrace, LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func ParseLevel(s string) Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "trace":
		return LevelTrace
	case "debug":
		return LevelDebug
	case "warn", "warning":
		return LevelWarn
	case "error", "fatal", "panic":
		return LevelError
	case "info", "":
		return LevelInfo
	default:
		return LevelInfo
	}
}

var (
	defaultWriter io.Writer       = os.Stdout
	std                           = log.New(defaultWriter, "", log.LstdFlags)
	mu            sync.RWMutex
	currentLevel  Level           = LevelInfo
	slogLevelVar  *slog.LevelVar  = new(slog.LevelVar)
	initOnce      sync.Once
)

func init() {
	slogLevelVar.Set(slog.LevelInfo)
	handler := slog.NewTextHandler(defaultWriter, &slog.HandlerOptions{
		Level: slogLevelVar,
	})
	slog.SetDefault(slog.New(handler))
}

// Init 初始化日志系统（设置日志级别与输出）
func Init(level string) {
	SetLevel(level)
}

// SetLevel 动态调整日志级别（线程安全，同时同步 slog）
func SetLevel(levelText string) {
	mu.Lock()
	defer mu.Unlock()

	parsed := ParseLevel(levelText)
	currentLevel = parsed
	slogLevelVar.Set(parsed.SlogLevel())
	slog.SetLogLoggerLevel(parsed.SlogLevel())
	std.SetPrefix("")
	std.SetFlags(log.LstdFlags)
}

// GetLevel 获取当前日志级别
func GetLevel() Level {
	mu.RLock()
	defer mu.RUnlock()
	return currentLevel
}

// IsDebug 检查当前是否开启了 Debug 或更低（Trace）日志级别
func IsDebug() bool {
	mu.RLock()
	defer mu.RUnlock()
	return currentLevel <= LevelDebug
}

func enabled(level Level) bool {
	mu.RLock()
	defer mu.RUnlock()
	return level >= currentLevel
}

func logf(level Level, format string, args ...interface{}) {
	if !enabled(level) {
		return
	}
	std.Printf("[%s] %s", level.String(), fmt.Sprintf(format, args...))
}

func Trace(format string, args ...interface{}) {
	logf(LevelTrace, format, args...)
}

func Debug(format string, args ...interface{}) {
	logf(LevelDebug, format, args...)
}

func Info(format string, args ...interface{}) {
	logf(LevelInfo, format, args...)
}

func Warn(format string, args ...interface{}) {
	logf(LevelWarn, format, args...)
}

func Error(format string, args ...interface{}) {
	logf(LevelError, format, args...)
}

func Fatal(format string, args ...interface{}) {
	logf(LevelError, format, args...)
	os.Exit(1)
}

func Fatalf(format string, args ...interface{}) {
	Fatal(format, args...)
}

func SetOutput(w io.Writer) {
	mu.Lock()
	defer mu.Unlock()
	std.SetOutput(w)
	handler := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: slogLevelVar,
	})
	slog.SetDefault(slog.New(handler))
}

func ResetOutput() {
	mu.Lock()
	defer mu.Unlock()
	std.SetOutput(defaultWriter)
	handler := slog.NewTextHandler(defaultWriter, &slog.HandlerOptions{
		Level: slogLevelVar,
	})
	slog.SetDefault(slog.New(handler))
}

