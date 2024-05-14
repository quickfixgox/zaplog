package zaplog

import (
	"fmt"
	"strings"

	"github.com/quickfixgo/quickfix"
	"github.com/quickfixgo/quickfix/config"
	"go.uber.org/zap/zapcore"
)

type zapLogFactoryOption func(*zapLogFactory)

type LogExtension string

const (
	LogExtension_Log   LogExtension = "log"
	LogExtension_Jsonl LogExtension = "jsonl"
	LogExtension_Plain LogExtension = "txt"
)

type loggerConfig struct {
	consoleLogLevel zapcore.Level
	CommonLogPath   string
	ErrorLogPath    string
	MaxSize         int // in megabytes
	MaxBackups      int
	MaxAge          int // in days
	Compress        bool
	extension       LogExtension
}

type zapLogFactory struct {
	globalLogPath   string
	sessionLogPaths map[quickfix.SessionID]string
	logCfg          loggerConfig
}

func (f zapLogFactory) Create() (quickfix.Log, error) {
	return newZapLog("GLOBAL", f.globalLogPath, f.logCfg)
}

func (f zapLogFactory) CreateSessionLog(sessionID quickfix.SessionID) (quickfix.Log, error) {
	logPath, ok := f.sessionLogPaths[sessionID]

	if !ok {
		return nil, fmt.Errorf("logger not defined for %v", sessionID)
	}

	prefix := sessionIDFilenamePrefix(sessionID)
	return newZapLog(prefix, logPath, f.logCfg)
}

func NewZapLogFactory(settings *quickfix.Settings, opts ...zapLogFactoryOption) (zapLogFactory, error) {
	logFactory := zapLogFactory{
		logCfg: loggerConfig{
			consoleLogLevel: zapcore.ErrorLevel,
			MaxSize:         2000, // megabytes
			MaxBackups:      5,
			MaxAge:          7, // days
			Compress:        false,
			extension:       "log",
		},
	}

	for _, opt := range opts {
		opt(&logFactory)
	}

	var err error
	if logFactory.globalLogPath, err = settings.GlobalSettings().Setting(config.FileLogPath); err != nil {
		return logFactory, err
	}

	logFactory.sessionLogPaths = make(map[quickfix.SessionID]string)

	for sid, sessionSettings := range settings.SessionSettings() {
		logPath, err := sessionSettings.Setting(config.FileLogPath)
		if err != nil {
			return logFactory, err
		}
		logFactory.sessionLogPaths[sid] = logPath
	}

	return logFactory, nil
}

func WithConsoleLogLevel(level zapcore.Level) zapLogFactoryOption {
	return func(f *zapLogFactory) {
		f.logCfg.consoleLogLevel = level
	}
}

func WithMaxSize(size int) zapLogFactoryOption {
	return func(f *zapLogFactory) {
		f.logCfg.MaxSize = size
	}
}

func WithMaxBackups(backups int) zapLogFactoryOption {
	return func(f *zapLogFactory) {
		f.logCfg.MaxBackups = backups
	}
}

func WithMaxAge(age int) zapLogFactoryOption {
	return func(f *zapLogFactory) {
		f.logCfg.MaxAge = age
	}
}

func WithCompress(compress bool) zapLogFactoryOption {
	return func(f *zapLogFactory) {
		f.logCfg.Compress = compress
	}
}

func WithExtension(extension LogExtension) zapLogFactoryOption {
	return func(f *zapLogFactory) {
		f.logCfg.extension = extension
	}
}

// copied from quickfix/session.go
func sessionIDFilenamePrefix(s quickfix.SessionID) string {
	sender := []string{s.SenderCompID}
	if s.SenderSubID != "" {
		sender = append(sender, s.SenderSubID)
	}
	if s.SenderLocationID != "" {
		sender = append(sender, s.SenderLocationID)
	}

	target := []string{s.TargetCompID}
	if s.TargetSubID != "" {
		target = append(target, s.TargetSubID)
	}
	if s.TargetLocationID != "" {
		target = append(target, s.TargetLocationID)
	}

	fname := []string{s.BeginString, strings.Join(sender, "_"), strings.Join(target, "_")}
	if s.Qualifier != "" {
		fname = append(fname, s.Qualifier)
	}
	return strings.Join(fname, "-")
}
