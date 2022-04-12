package logger

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type Logger interface {
	Debug(msg string)
	Debugf(fmt string, args ...any)
	Info(msg string)
	Infof(fmt string, args ...any)
	Warn(msg string)
	Warnf(fmt string, args ...any)
	Error(msg string)
	Errorf(fmt string, args ...any)
	Fatal(msg string)
	Fatalf(fmt string, args ...any)
	GetLevel() string
}

// CustomLogger описывает логгер приложения.
type CustomLogger struct {
	logLevel string
	logger   *zerolog.Logger
}

const logLevelEnvName = "LOG_LEVEL"

var (
	logger *CustomLogger
	once   sync.Once
)

// GetLogger is getter to create new logger instance.
func GetLogger() *CustomLogger {
	once.Do(func() {
		// keep err because on linux, android, netbsd, dragonfly err is nil.
		// Otherwise useFullPath is false.
		logger = newLogger(strings.ToLower(os.Getenv(logLevelEnvName)))
	})

	return logger
}

func newLogger(logLevel string) *CustomLogger {
	customLog := &CustomLogger{}
	if logLevel == "" {
		customLog.logLevel = zerolog.InfoLevel.String()
	} else {
		customLog.logLevel = logLevel
	}

	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	output.FormatTimestamp = func(i any) string {
		return fmt.Sprintf("ts=%s", i)
	}
	output.FormatLevel = func(i any) string {
		return fmt.Sprintf("lvl=%s", strings.ToUpper(i.(string))) // nolint:forcetypeassert
	}
	output.FormatMessage = func(i any) string {
		str, _ := i.(string)

		return fmt.Sprintf("msg=\"%s\"", editStringWithQuotes(str))
	}
	output.FormatFieldName = func(i any) string {
		return fmt.Sprintf("%s=", i)
	}
	output.FormatFieldValue = func(i any) string {
		return fmt.Sprintf("\"%s\"", i)
	}

	level, err := zerolog.ParseLevel(customLog.logLevel)
	if err != nil {
		zlog := zerolog.New(output).With().Timestamp().Logger()
		zlog.Fatal().Str("src", "logger").Msgf("can't create logger: %s", err)
	}

	zlog := zerolog.New(output).With().Timestamp().Logger().Level(level)
	customLog.logger = &zlog

	return customLog
}

func (l *CustomLogger) Debug(msg string) {
	l.logger.Debug().Str("src", getSource()).Msg(msg)
}

func (l *CustomLogger) Debugf(fmt string, args ...any) {
	l.logger.Debug().Str("src", getSource()).Msgf(fmt, args...)
}

func (l *CustomLogger) Info(msg string) {
	l.logger.Info().Str("src", getSource()).Msg(msg)
}

func (l *CustomLogger) Infof(fmt string, args ...any) {
	l.logger.Info().Str("src", getSource()).Msgf(fmt, args...)
}

func (l *CustomLogger) Warn(msg string) {
	l.logger.Warn().Str("src", getSource()).Msg(msg)
}

func (l *CustomLogger) Warnf(fmt string, args ...any) {
	l.logger.Warn().Str("src", getSource()).Msgf(fmt, args...)
}

func (l *CustomLogger) Error(msg string) {
	l.logger.Error().Str("src", getSource()).Msg(msg)
}

func (l *CustomLogger) Errorf(fmt string, args ...any) {
	l.logger.Error().Str("src", getSource()).Msgf(fmt, args...)
}

func (l *CustomLogger) Fatal(msg string) {
	l.logger.Fatal().Str("src", getSource()).Msg(msg)
}

func (l *CustomLogger) Fatalf(fmt string, args ...any) {
	l.logger.Fatal().Str("src", getSource()).Msgf(fmt, args...)
}

func (l *CustomLogger) GetLevel() string {
	return l.logger.GetLevel().String()
}

func getSource() string {
	const stackDepth = 2

	_, file, line, ok := runtime.Caller(stackDepth)
	if !ok {
		return "unknown"
	}

	filePrefix := file[strings.LastIndex(file, "/")+1:]

	filename := filePrefix + ":" + strconv.Itoa(line)

	return filename
}

func editStringWithQuotes(stringWithQuotes string) string {
	return strings.ReplaceAll(stringWithQuotes, "\"", "\\\"")
}
