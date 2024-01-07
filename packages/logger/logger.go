package logger

import (
    "fmt"
    "github.com/hashicorp/go-hclog"
    "io"
    "os"
)

var logger hclog.Logger
var logFile *os.File
var logLevel hclog.Level

func InitLogger(logDirPath, logLevelController string) error {
    logFileName := "iterator.log"
    logFilePath := logDirPath + "/" + logFileName

    logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    if err != nil {
        return fmt.Errorf("Failed to open log file at path %s: %v", logFilePath, err)
    }

    multiWriter := io.MultiWriter(os.Stdout, logFile)


    switch logLevelController {
    case "debug":
        logLevel = hclog.Debug
    case "info":
        logLevel = hclog.Info
    case "warn":
        logLevel = hclog.Warn
    case "error":
        logLevel = hclog.Error
    case "fatal":
        logLevel = hclog.Error
    default:
        logLevel = hclog.Info
    }
    logger = hclog.New(&hclog.LoggerOptions{
        Name:   "ITERATOR",
        Level:  logLevel,
        Output: multiWriter,
    })

    return nil
}

func CloseLogger() {
    if logFile != nil {
        logFile.Close()
    }
}

func Debug(format string, v ...interface{}) {
    formattedMessage := fmt.Sprintf(format, v...)
    logger.Debug(formattedMessage)
}

func Info(format string, v ...interface{}) {
    formattedMessage := fmt.Sprintf(format, v...)
    logger.Info(formattedMessage)
}

func Warn(format string, v ...interface{}) {
    formattedMessage := fmt.Sprintf(format, v...)
    logger.Warn(formattedMessage)
}

func Error(format string, v ...interface{}) {
    formattedMessage := fmt.Sprintf(format, v...)
    logger.Error(formattedMessage)
}

func Fatal(format string, v ...interface{}) {
    formattedMessage := fmt.Sprintf(format, v...)
    logger.Error("FATAL: "+formattedMessage)
    os.Exit(1)
}
