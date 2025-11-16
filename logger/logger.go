package logger

import (
	"log"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	Info  *log.Logger
	Error *log.Logger
	Debug *log.Logger
)

// InitLogger sets up logging to file with automatic rotation
func InitLogger(logFilePath string) {
	// Create logs directory if it doesn't exist
	if err := os.MkdirAll("logs", 0755); err != nil {
		log.Fatal("Failed to create logs directory:", err)
	}

	// Set up log rotation
	logFile := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    10,   // megabytes
		MaxBackups: 3,    // number of backups to keep
		MaxAge:     28,   // days
		Compress:   true, // compress old log files
	}

	// Initialize loggers with different prefixes (writing only to file)
	Info = log.New(logFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(logFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	Debug = log.New(logFile, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
}
