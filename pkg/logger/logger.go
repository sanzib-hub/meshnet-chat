package logger

import (
	"fmt"
	"io"
	"log"
	"os"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

type Logger struct {
	level  Level
	logger *log.Logger
}

var defaultLogger = New(LevelInfo, os.Stdout)

func New(level Level, out io.Writer) *Logger {
	return &Logger{
		level:  level,
		logger: log.New(out, "", log.LstdFlags),
	}
}

func (l *Logger) log(level Level, prefix, format string, args ...interface{}) {
	if level < l.level {
		return
	}
	l.logger.Output(3, fmt.Sprintf("["+prefix+"] "+format, args...))
}

func (l *Logger) Debug(format string, args ...interface{}) { l.log(LevelDebug, "DEBUG", format, args...) }
func (l *Logger) Info(format string, args ...interface{})  { l.log(LevelInfo, "INFO", format, args...) }
func (l *Logger) Warn(format string, args ...interface{})  { l.log(LevelWarn, "WARN", format, args...) }
func (l *Logger) Error(format string, args ...interface{}) { l.log(LevelError, "ERROR", format, args...) }

// Package-level helpers using the default logger.
func Debug(format string, args ...interface{}) { defaultLogger.Debug(format, args...) }
func Info(format string, args ...interface{})  { defaultLogger.Info(format, args...) }
func Warn(format string, args ...interface{})  { defaultLogger.Warn(format, args...) }
func Error(format string, args ...interface{}) { defaultLogger.Error(format, args...) }

func SetLevel(level Level) { defaultLogger.level = level }
