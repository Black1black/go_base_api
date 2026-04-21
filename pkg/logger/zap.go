package logger

import (
	"runtime"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// asyncLogger реализует Logger интерфейс
type asyncLogger struct {
	*zap.Logger
	ch      chan func()   // Канал для асинхронной обработки
	stop    chan struct{} // Канал для остановки
	running bool          // Флаг работы
	once    sync.Once     // Гарантирует однократное закрытие
}

func New(level string, bufferSize int) Logger {
	config := zap.NewProductionConfig()
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder // Формат времени
	config.EncoderConfig.TimeKey = "timestamp"                   // Ключ для времени

	var zapLevel zapcore.Level
	switch strings.ToLower(level) {
	case "debug":
		zapLevel = zapcore.DebugLevel
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}
	config.Level = zap.NewAtomicLevelAt(zapLevel)

	zapLogger, _ := config.Build(
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.AddCallerSkip(1),
	)

	l := &asyncLogger{
		Logger:  zapLogger,
		ch:      make(chan func(), bufferSize),
		stop:    make(chan struct{}),
		running: true,
	}

	go l.processor()

	return l
}

func (l *asyncLogger) processor() {
	defer func() {
		if r := recover(); r != nil {
			l.Logger.Error("Logger panic recovered", zap.Any("error", r))
		}
	}()

	for {
		select {
		case task := <-l.ch:
			task()
		case <-l.stop:
			return
		}
	}
}

func getCallerInfo(skip int) (file string, function string, line int) {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown", "unknown", 0
	}

	if fn := runtime.FuncForPC(pc); fn != nil {
		parts := strings.Split(fn.Name(), ".")
		function = parts[len(parts)-1]
	}

	if fileParts := strings.Split(file, "/"); len(fileParts) > 3 {
		file = strings.Join(fileParts[len(fileParts)-3:], "/")
	}

	return file, function, line
}

func (l *asyncLogger) log(level zapcore.Level, msg string, fields []zap.Field) {
	if !l.running {
		return
	}

	l.ch <- func() {
		switch level {
		case zapcore.DebugLevel:
			l.Logger.Debug(msg, fields...)
		case zapcore.InfoLevel:
			l.Logger.Info(msg, fields...)
		case zapcore.WarnLevel:
			l.Logger.Warn(msg, fields...)
		case zapcore.ErrorLevel:
			l.Logger.Error(msg, fields...)
		}
	}
}

func (l *asyncLogger) Debug(msg string, fields ...any) {
	l.log(zapcore.DebugLevel, msg, convertFields(fields))
}

func (l *asyncLogger) Info(msg string, fields ...any) {
	l.log(zapcore.InfoLevel, msg, convertFields(fields))
}

func (l *asyncLogger) Warn(msg string, fields ...any) {
	l.log(zapcore.WarnLevel, msg, convertFields(fields))
}

func (l *asyncLogger) Error(msg string, err error, fields ...any) {
	if !l.running {
		return
	}

	file, function, line := getCallerInfo(2)

	zapFields := []zap.Field{
		zap.Error(err),
		zap.String("file", file),
		zap.String("func", function),
		zap.Int("line", line),
	}

	zapFields = append(zapFields, convertFields(fields)...)

	l.log(zapcore.ErrorLevel, msg, zapFields)
}

// Sync корректно останавливает логгер
func (l *asyncLogger) Sync() error {
	l.once.Do(func() {
		l.running = false
		close(l.stop)
	})
	return l.Logger.Sync()
}

// Преобразует any в zap.Field
func convertFields(fields []any) []zap.Field {
	zapFields := make([]zap.Field, 0, len(fields)/2)
	for i := 0; i < len(fields); i += 2 {
		if i+1 >= len(fields) {
			continue
		}
		if key, ok := fields[i].(string); ok {
			zapFields = append(zapFields, zap.Any(key, fields[i+1]))
		}
	}
	return zapFields
}
