package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

func SetupLogger() (*slog.Logger, func()) {
	absLogsDir, err := filepath.Abs("logs")
	if err != nil {
		panic("failed to get absolute path for logs directory: " + err.Error())
	}

	if err := os.MkdirAll(absLogsDir, 0o755); err != nil {
		panic("failed to create logs directory at " + absLogsDir + ": " + err.Error())
	}

	logFileName := filepath.Join(absLogsDir, "marketflow_"+time.Now().Format("20060102_150405")+".log")
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		panic("failed to open log file at " + logFileName + ": " + err.Error())
	}

	multiWriter := io.MultiWriter(os.Stdout, logFile)
	logger := slog.New(slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
		Level: slog.LevelInfo, // Changed from Debug to Info to reduce verbosity
	}))

	cleanup := func() {
		if err := logFile.Close(); err != nil {
			logger.Error("failed to close log file", "error", err)
		}
	}

	logger.Info("Logger initialized",
		"logFile", logFileName,
		"absPath", absLogsDir)
	return logger, cleanup
}
