package zaplog

import (
	"fmt"
	"os"
	"path"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type zapLog struct {
	eventLogger   *zap.SugaredLogger
	messageLogger *zap.SugaredLogger
}

func (l zapLog) OnIncoming(s []byte) {
	l.messageLogger.Infof("[INCOMING <-]:  %s", s)
}

func (l zapLog) OnOutgoing(s []byte) {
	l.messageLogger.Infof("[OUTGOING ->]: %s", s)
}

func (l zapLog) OnEvent(s string) {
	l.eventLogger.Infof("%s", s)
}

func (l zapLog) OnEventf(format string, a ...interface{}) {
	l.eventLogger.Infof(format, a...)
}

func newZapLog(prefix string, logPath string, logCfg loggerConfig) (zapLog, error) {
	l := zapLog{}

	eventFileName := fmt.Sprintf("%s.event.current.%s", prefix, logCfg.extension)
	messageFileName := fmt.Sprintf("%s.messages.current.%s", prefix, logCfg.extension)

	eventLogName := path.Join(logPath, eventFileName)
	messageLogName := path.Join(logPath, messageFileName)

	eventWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   eventLogName,
		MaxSize:    logCfg.MaxSize, // megabytes
		MaxBackups: logCfg.MaxBackups,
		MaxAge:     logCfg.MaxAge, // days
		Compress:   logCfg.Compress,
	})
	messageWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   messageLogName,
		MaxSize:    logCfg.MaxSize, // megabytes
		MaxBackups: logCfg.MaxBackups,
		MaxAge:     logCfg.MaxAge, // days
		Compress:   logCfg.Compress,
	})
	consoleWriter := zapcore.AddSync(os.Stdout)

	pe := zap.NewProductionEncoderConfig()
	pe.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoder := zapcore.NewJSONEncoder(pe)

	pe.EncodeLevel = zapcore.CapitalColorLevelEncoder // colorize log level
	consoleEncoder := zapcore.NewConsoleEncoder(pe)

	eventCore := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, eventWriter, zap.InfoLevel),
		zapcore.NewCore(consoleEncoder, consoleWriter, logCfg.consoleLogLevel),
	)
	messageCore := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, messageWriter, zap.InfoLevel),
		zapcore.NewCore(consoleEncoder, consoleWriter, logCfg.consoleLogLevel),
	)

	l.eventLogger = zap.New(eventCore, zap.AddCaller(), zap.AddCallerSkip(1)).Sugar()
	l.messageLogger = zap.New(messageCore, zap.AddCaller(), zap.AddCallerSkip(1)).Sugar()

	return l, nil
}
