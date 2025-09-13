package util

import (
	"io"
	"log"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

type LoggerConfig struct {
	LogFile    string
	MaxSizeMB  int
	MaxBackups int
	MaxAgeDays int
	LogToFile  bool
}

func SetupLogger(cfg LoggerConfig) *log.Logger {
	println("Setting up logger...")
	var writer io.Writer
	if cfg.LogToFile {
		writer = &lumberjack.Logger{
			Filename:   "app.log",
			MaxSize:    5, // megabytes
			MaxBackups: 3,
			Compress:   true,
		}
	} else {
		writer = os.Stdout
	}
	return log.New(writer, "", log.LstdFlags|log.Lshortfile)
}
