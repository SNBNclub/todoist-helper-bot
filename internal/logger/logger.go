package logger

import (
	"bytes"
	"fmt"
	"net/http"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger

func init() {
	Log = getLogger()
}

func getLogger() *zap.Logger {
	consoleEncoder := zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
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

	consoleLogginLevel := zap.NewAtomicLevelAt(zapcore.DebugLevel)
	botLogginLevel := zap.NewAtomicLevelAt(zapcore.PanicLevel)
	fileLoggingLevel := zap.NewAtomicLevelAt(zapcore.DebugLevel)

	http.HandleFunc("/logging_level_console", consoleLogginLevel.ServeHTTP)
	http.HandleFunc("/loggin_level_bot", botLogginLevel.ServeHTTP)
	http.HandleFunc("/logging_level_file", fileLoggingLevel.ServeHTTP)

	logFile, err := os.OpenFile("/var/log/tracker.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0777)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
	}

	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, os.Stderr, consoleLogginLevel),
		zapcore.NewCore(jsonEncoder, zapcore.AddSync(httpSyncer), botLogginLevel),
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(logFile), fileLoggingLevel),
	)

	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))
	return logger
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
