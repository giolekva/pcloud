package log

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	// LevelDebug represents very verbose messages for debugging specific issues
	LevelDebug = "debug"
	// LevelInfo represents default log level, informational
	LevelInfo = "info"
	// LevelWarn represents messages about possible issues
	LevelWarn = "warn"
	// LevelError represents messages about things we know are problems
	LevelError = "error"
)

// Type and function aliases from zap to limit the libraries scope into our code
type Field = zapcore.Field

var Int = zap.Int
var String = zap.String
var Any = zap.Any
var Err = zap.Error
var Time = zap.Time
var Duration = zap.Duration

// Logger logs messages
type Logger struct {
	zap          *zap.Logger
	consoleLevel zap.AtomicLevel
	fileLevel    zap.AtomicLevel
}

// LoggerConfiguration represents configuration of the logger
type LoggerConfiguration struct {
	EnableConsole bool
	ConsoleJSON   bool
	ConsoleLevel  string
	EnableFile    bool
	FileJSON      bool
	FileLevel     string
	FileLocation  string
}

// NewLogger creates new logger
func NewLogger(config *LoggerConfiguration) *Logger {
	cores := []zapcore.Core{}
	logger := &Logger{
		consoleLevel: zap.NewAtomicLevelAt(getZapLevel(config.ConsoleLevel)),
		fileLevel:    zap.NewAtomicLevelAt(getZapLevel(config.FileLevel)),
	}

	if config.EnableConsole {
		writer := zapcore.Lock(os.Stderr)
		core := zapcore.NewCore(makeEncoder(config.ConsoleJSON), writer, logger.consoleLevel)
		cores = append(cores, core)
	}

	if config.EnableFile {
		writer := zapcore.AddSync(&lumberjack.Logger{
			Filename: config.FileLocation,
			MaxSize:  100,
			Compress: true,
		})
		core := zapcore.NewCore(makeEncoder(config.FileJSON), writer, logger.fileLevel)
		cores = append(cores, core)
	}

	combinedCore := zapcore.NewTee(cores...)

	logger.zap = zap.New(combinedCore,
		zap.AddCaller(),
	)

	return logger
}

func (l *Logger) Debug(message string, fields ...Field) {
	l.zap.Debug(message, fields...)
}

func (l *Logger) Info(message string, fields ...Field) {
	l.zap.Info(message, fields...)
}

func (l *Logger) Warn(message string, fields ...Field) {
	l.zap.Warn(message, fields...)
}

func (l *Logger) Error(message string, fields ...Field) {
	l.zap.Error(message, fields...)
}

func getZapLevel(level string) zapcore.Level {
	switch level {
	case LevelInfo:
		return zapcore.InfoLevel
	case LevelWarn:
		return zapcore.WarnLevel
	case LevelDebug:
		return zapcore.DebugLevel
	case LevelError:
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func makeEncoder(json bool) zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	if json {
		return zapcore.NewJSONEncoder(encoderConfig)
	}

	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	return zapcore.NewConsoleEncoder(encoderConfig)
}
