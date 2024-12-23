package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

var (
	infoLogger    *log.Logger
	warningLogger *log.Logger
	errorLogger   *log.Logger
	debugLogger   *log.Logger
)

// 初始化日志记录器
func init() {
	// 创建日志目录
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatal("创建日志目录失败:", err)
	}

	// 创建或打开日志文件
	currentTime := time.Now()
	logFileName := filepath.Join(logDir, fmt.Sprintf("%s.log", currentTime.Format("2006-01-02")))
	logFile, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("打开日志文件失败:", err)
	}

	// 设置日志格式
	flags := log.Ldate | log.Ltime | log.Lmicroseconds

	// 初始化不同级别的日志记录器
	infoLogger = log.New(logFile, "[INFO] ", flags)
	warningLogger = log.New(logFile, "[WARN] ", flags)
	errorLogger = log.New(logFile, "[ERROR] ", flags)
	debugLogger = log.New(logFile, "[DEBUG] ", flags)
}

// getFileAndLine 获取调用者的文件名和行号
func getFileAndLine() string {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return "unknown:0"
	}
	shortFile := filepath.Base(file)
	return fmt.Sprintf("%s:%d", shortFile, line)
}

// formatMessage 格式化日志消息
func formatMessage(format string, args ...interface{}) string {
	location := getFileAndLine()
	message := fmt.Sprintf(format, args...)
	return fmt.Sprintf("[%s] %s", location, message)
}

// Info 记录信息级别的日志
func Info(format string, args ...interface{}) {
	infoLogger.Printf(formatMessage(format, args...))
}

// Warn 记录警告级别的日志
func Warn(format string, args ...interface{}) {
	warningLogger.Printf(formatMessage(format, args...))
}

// Error 记录错误级别的日志
func Error(format string, args ...interface{}) {
	errorLogger.Printf(formatMessage(format, args...))
}

// Debug 记录调试级别的日志
func Debug(format string, args ...interface{}) {
	debugLogger.Printf(formatMessage(format, args...))
}

// Fatal 记录致命错误并退出程序
func Fatal(format string, args ...interface{}) {
	errorLogger.Printf(formatMessage(format, args...))
	os.Exit(1)
}
