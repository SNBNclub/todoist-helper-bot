package logger

import (
	"bytes"
	"fmt"
	"net/http"
	"os"

	"example.com/bot/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var builder *loggerBuilder

func NewLogger(cfg *config.LoggerConfig) (*zap.Logger, error) {
	if builder == nil {
		builder = &loggerBuilder{loggerConfig: cfg}
		return builder.get()
	}
	return nil, fmt.Errorf("logger instance already created")
}

type loggerBuilder struct {
	consoleFileEncoder        zapcore.Encoder
	consoleFileEncoderCreated bool
	loggerConfig              *config.LoggerConfig
}

func (b *loggerBuilder) get() (logger *zap.Logger, err error) {
	cores := make([]zapcore.Core, 0, 3)
	if b.loggerConfig.ENABLE_CONSOLE_LOGGER {
		cores = append(cores, b.getConsoleLoggerCore())
	}
	if b.loggerConfig.ENABLE_FILE_LOGGER {
		core, errGet := b.getFileLoggerCore()
		if errGet != nil {
			err = fmt.Errorf("unable to create file logger core: %w", errGet)
		} else {
			cores = append(cores, core)
		}
	}
	if b.loggerConfig.ENABLE_BOT_LOGGER {
		cores = append(cores, b.getBotLoggerCore())
	}
	if len(cores) == 0 {
		return nil, fmt.Errorf("unable to create logger, all cores failed: %w", err)
	}
	core := zapcore.NewTee(cores...)
	logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))
	return
}

func (b *loggerBuilder) createConsoleFileEncoder() {
	if b.consoleFileEncoderCreated {
		return
	}
	b.consoleFileEncoder = zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
		MessageKey:     "message",
		LevelKey:       "level",
		TimeKey:        "time",
		NameKey:        "logger",
		CallerKey:      "caller",
		StacktraceKey:  "stacktrace",
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	})
}

func (b *loggerBuilder) getConsoleLoggerCore() zapcore.Core {
	b.createConsoleFileEncoder()
	consoleLogginLevel := zap.NewAtomicLevelAt(zapcore.DebugLevel)
	http.HandleFunc("/logging_level_console", consoleLogginLevel.ServeHTTP)
	return zapcore.NewCore(b.consoleFileEncoder, os.Stderr, consoleLogginLevel)
}

func (b *loggerBuilder) getFileLoggerCore() (zapcore.Core, error) {
	b.createConsoleFileEncoder()
	fileLoggingLevel := zap.NewAtomicLevelAt(zapcore.DebugLevel)
	http.HandleFunc("/logging_level_file", fileLoggingLevel.ServeHTTP)
	logFile, err := os.OpenFile("/var/log/tracker.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0777)
	if err != nil {
		return nil, fmt.Errorf("unable to open log file: %w", err)
	}
	return zapcore.NewCore(b.consoleFileEncoder, zapcore.AddSync(logFile), fileLoggingLevel), nil
}

func (b *loggerBuilder) getBotLoggerCore() zapcore.Core {
	jsonEncoder := zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		MessageKey:     "message",
		LevelKey:       "level",
		TimeKey:        "time",
		NameKey:        "logger",
		CallerKey:      "caller",
		StacktraceKey:  "stacktrace",
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	})
	httpSyncer := &writeSyncerHTTP{
		URL: "http://127.0.0.1:5040/logging/tracker",
	}
	botLogginLevel := zap.NewAtomicLevelAt(zapcore.PanicLevel)
	http.HandleFunc("/loggin_level_bot", botLogginLevel.ServeHTTP)
	return zapcore.NewCore(jsonEncoder, zapcore.AddSync(httpSyncer), botLogginLevel)
}

type writeSyncerHTTP struct {
	URL string
}

func (w *writeSyncerHTTP) Write(p []byte) (n int, err error) {
	resp, err := http.Post(w.URL, "application/json", bytes.NewBuffer(p))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}
	return len(p), nil
}

func (w *writeSyncerHTTP) Sync() error {
	return nil
}
