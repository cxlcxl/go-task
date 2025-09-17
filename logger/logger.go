package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

type Logger struct {
	*log.Logger
	logPath string
}

var defaultLogger *Logger

func Init(logPath string) error {
	if err := os.MkdirAll(logPath, 0755); err != nil {
		return err
	}

	logFile := filepath.Join(logPath, fmt.Sprintf("executor-%s.log", time.Now().Format("2006-01-02")))
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	multiWriter := io.MultiWriter(os.Stdout, file)
	defaultLogger = &Logger{
		Logger:  log.New(multiWriter, "", log.LstdFlags|log.Lshortfile),
		logPath: logPath,
	}

	return nil
}

func Info(format string, v ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Printf("[INFO] "+format, v...)
	}
}

func Error(format string, v ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Printf("[ERROR] "+format, v...)
	}
}

func Debug(format string, v ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Printf("[DEBUG] "+format, v...)
	}
}

func Warn(format string, v ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Printf("[WARN] "+format, v...)
	}
}

// JobLogger creates a job-specific logger
func JobLogger(logID int, logDateTime time.Time) *Logger {
	if defaultLogger == nil {
		return nil
	}

	logFile := filepath.Join(defaultLogger.logPath, fmt.Sprintf("job-%d-%s.log", logID, logDateTime.Format("2006-01-02")))
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		Error("Failed to create job log file: %v", err)
		return defaultLogger
	}

	return &Logger{
		Logger:  log.New(file, "", log.LstdFlags),
		logPath: defaultLogger.logPath,
	}
}

func (l *Logger) JobInfo(format string, v ...interface{}) {
	l.Printf("[JOB-INFO] "+format, v...)
}

func (l *Logger) JobError(format string, v ...interface{}) {
	l.Printf("[JOB-ERROR] "+format, v...)
}
