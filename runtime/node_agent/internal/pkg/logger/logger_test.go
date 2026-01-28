package logger

import (
	"testing"
)

// TestLogLevel 测试日志级别
func TestLogLevel(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{LevelDebug, "debug"},
		{LevelInfo, "info"},
		{LevelWarn, "warn"},
		{LevelError, "error"},
	}

	for _, tt := range tests {
		if string(tt.level) != tt.expected {
			t.Errorf("LogLevel %s 期望 %s，实际 %s", tt.level, tt.expected, string(tt.level))
		}
	}
}

// TestNewLogger 测试创建日志器
func TestNewLogger(t *testing.T) {
	logger := NewSimpleLogger(LevelInfo)
	if logger == nil {
		t.Error("NewSimpleLogger 不应返回 nil")
	}
	if logger.level != LevelInfo {
		t.Errorf("日志级别期望 %s，实际 %s", LevelInfo, logger.level)
	}
}

// TestSimpleLogger_LevelComparison 测试日志级别比较
func TestSimpleLogger_LevelComparison(t *testing.T) {
	tests := []struct {
		loggerLevel LogLevel
		msgLevel    LogLevel
		expected    bool
	}{
		{LevelDebug, LevelDebug, true},
		{LevelDebug, LevelInfo, true},
		{LevelDebug, LevelError, true},
		{LevelInfo, LevelDebug, false},
		{LevelInfo, LevelInfo, true},
		{LevelInfo, LevelError, true},
		{LevelError, LevelDebug, false},
		{LevelError, LevelInfo, false},
		{LevelError, LevelError, true},
	}

	for _, tt := range tests {
		logger := NewSimpleLogger(tt.loggerLevel)
		result := logger.shouldLog(tt.msgLevel)
		if result != tt.expected {
			t.Errorf("shouldLog(%s, %s) 期望 %v，实际 %v",
				tt.loggerLevel, tt.msgLevel, tt.expected, result)
		}
	}
}

// TestSimpleLogger_LogMethods 测试日志方法不 panic
func TestSimpleLogger_LogMethods(t *testing.T) {
	logger := NewSimpleLogger(LevelDebug)

	// 测试所有日志级别方法
	logger.Debug("debug message", "key", "value")
	logger.Info("info message", "key", "value")
	logger.Warn("warn message", "key", "value")
	logger.Error("error message", "key", "value")
}

// TestGlobalLogger 测试全局日志实例
func TestGlobalLogger(t *testing.T) {
	if GlobalLogger == nil {
		t.Error("GlobalLogger 不应为 nil")
	}

	// 测试全局日志实例可以正常工作
	GlobalLogger.Info("测试全局日志")
	GlobalLogger.Debug("调试信息")
	GlobalLogger.Warn("警告信息")
	GlobalLogger.Error("错误信息")
}
