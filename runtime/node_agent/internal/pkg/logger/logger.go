package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// LogLevel 日志级别
type LogLevel string

const (
	LevelDebug LogLevel = "debug"
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
)

// Logger 日志接口
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Close()
}

// FileLogger 文件日志实现
type FileLogger struct {
	level       LogLevel
	file        *os.File
	mu          sync.Mutex
	levelMap    map[LogLevel]int
	logDir      string
	retention   int // 保留天数
	lastCleanup time.Time
}

// 日志配置
const (
	DefaultRetention = 7                // 默认保留7天
	CleanupInterval  = 24 * time.Hour   // 每天检查一次
	MaxLogSize       = 10 * 1024 * 1024 // 单个日志文件最大10MB
)

// NewFileLogger 创建文件日志器
// level: 日志级别
// logDir: 日志目录
// retentionDays: 保留天数（默认7天）
func NewFileLogger(level LogLevel, logDir string, retentionDays ...int) (*FileLogger, error) {
	// 确保日志目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %w", err)
	}

	// 确定保留天数
	retention := DefaultRetention
	if len(retentionDays) > 0 && retentionDays[0] > 0 {
		retention = retentionDays[0]
	}

	logger := &FileLogger{
		level:       level,
		logDir:      logDir,
		retention:   retention,
		lastCleanup: time.Now(),
		levelMap: map[LogLevel]int{
			LevelDebug: 0,
			LevelInfo:  1,
			LevelWarn:  2,
			LevelError: 3,
		},
	}

	// 打开日志文件
	if err := logger.openLogFile(); err != nil {
		return nil, err
	}

	// 清理旧日志
	logger.cleanupOldLogs()

	return logger, nil
}

// openLogFile 打开当天的日志文件
func (l *FileLogger) openLogFile() error {
	// 生成日志文件名
	logFile := filepath.Join(l.logDir, fmt.Sprintf("agent_%s.log", time.Now().Format("2006-01-02")))

	// 打开日志文件（追加模式）
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}

	l.file = file
	return nil
}

// shouldRotate 检查是否需要轮转日志
func (l *FileLogger) shouldRotate() bool {
	// 检查日期是否变化
	currentFile := l.file.Name()
	currentDate := time.Now().Format("2006-01-02")
	expectedFile := filepath.Join(l.logDir, fmt.Sprintf("agent_%s.log", currentDate))

	return currentFile != expectedFile
}

// rotateIfNeeded 检查并轮转日志
func (l *FileLogger) rotateIfNeeded() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.shouldRotate() {
		// 关闭当前文件
		if l.file != nil {
			l.file.Close()
		}

		// 打开新文件
		l.openLogFile()

		// 清理旧日志
		l.cleanupOldLogs()
	}
}

// cleanupOldLogs 清理过期日志
func (l *FileLogger) cleanupOldLogs() {
	// 检查时间间隔
	if time.Since(l.lastCleanup) < CleanupInterval {
		return
	}

	l.lastCleanup = time.Now()

	// 计算截止日期
	cutoffDate := time.Now().AddDate(0, 0, -l.retention)

	// 读取日志目录
	entries, err := os.ReadDir(l.logDir)
	if err != nil {
		log.Printf("[Logger] 读取日志目录失败: %v", err)
		return
	}

	// 删除过期文件
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// 检查文件名格式
		name := entry.Name()
		if !strings.HasPrefix(name, "agent_") || !strings.HasSuffix(name, ".log") {
			continue
		}

		// 提取日期
		dateStr := strings.TrimPrefix(name, "agent_")
		dateStr = strings.TrimSuffix(dateStr, ".log")

		// 解析日期
		fileDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		// 如果文件日期早于截止日期，则删除
		if fileDate.Before(cutoffDate) {
			filePath := filepath.Join(l.logDir, name)
			if err := os.Remove(filePath); err != nil {
				log.Printf("[Logger] 删除旧日志失败: %v", err)
			} else {
				log.Printf("[Logger] 已清理旧日志: %s", name)
			}
		}
	}
}

// shouldLog 判断是否应该记录该级别的日志
func (l *FileLogger) shouldLog(level LogLevel) bool {
	return l.levelMap[level] >= l.levelMap[l.level]
}

// format 格式化日志消息
func (l *FileLogger) format(level LogLevel, msg string, args ...interface{}) string {
	// 获取调用者信息
	pc, file, line, _ := runtime.Caller(2)
	funcName := runtime.FuncForPC(pc).Name()
	funcName = extractFuncName(funcName)

	// 格式化消息
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}

	// 格式化时间
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")

	return fmt.Sprintf("[%s] [%s] [%s:%d %s] %s",
		timestamp,
		strings.ToUpper(string(level)),
		filepath.Base(file),
		line,
		funcName,
		msg)
}

// extractFuncName 从完整函数名中提取简洁名称
func extractFuncName(fullName string) string {
	parts := strings.Split(fullName, ".")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return fullName
}

// log 写入日志
func (l *FileLogger) log(level LogLevel, msg string, args ...interface{}) {
	if !l.shouldLog(level) {
		return
	}

	// 检查是否需要轮转日志
	l.rotateIfNeeded()

	formatted := l.format(level, msg, args...)

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		fmt.Fprintln(l.file, formatted)
	}
}

// Debug 记录 Debug 级别日志
func (l *FileLogger) Debug(msg string, args ...interface{}) {
	l.log(LevelDebug, msg, args...)
}

// Info 记录 Info 级别日志
func (l *FileLogger) Info(msg string, args ...interface{}) {
	l.log(LevelInfo, msg, args...)
}

// Warn 记录 Warn 级别日志
func (l *FileLogger) Warn(msg string, args ...interface{}) {
	l.log(LevelWarn, msg, args...)
}

// Error 记录 Error 级别日志
func (l *FileLogger) Error(msg string, args ...interface{}) {
	l.log(LevelError, msg, args...)
}

// Close 关闭日志文件
func (l *FileLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// SimpleLogger 控制台日志实现（用于交互模式显示）
type SimpleLogger struct {
	level LogLevel
}

func NewSimpleLogger(level LogLevel) *SimpleLogger {
	return &SimpleLogger{level: level}
}

func (l *SimpleLogger) shouldLog(level LogLevel) bool {
	levels := map[LogLevel]int{
		LevelDebug: 0,
		LevelInfo:  1,
		LevelWarn:  2,
		LevelError: 3,
	}
	return levels[level] >= levels[l.level]
}

func (l *SimpleLogger) log(level LogLevel, msg string, args ...interface{}) {
	if l.shouldLog(level) {
		timestamp := time.Now().Format("15:04:05")
		prefix := strings.ToUpper(string(level))
		if len(args) > 0 {
			msg = fmt.Sprintf(msg, args...)
		}
		fmt.Printf("[%s] [%s] %s\n", timestamp, prefix, msg)
	}
}

func (l *SimpleLogger) Debug(msg string, args ...interface{}) {
	l.log(LevelDebug, msg, args...)
}

func (l *SimpleLogger) Info(msg string, args ...interface{}) {
	l.log(LevelInfo, msg, args...)
}

func (l *SimpleLogger) Warn(msg string, args ...interface{}) {
	l.log(LevelWarn, msg, args...)
}

func (l *SimpleLogger) Error(msg string, args ...interface{}) {
	l.log(LevelError, msg, args...)
}

// GlobalLogger 全局日志实例（向后兼容）
var GlobalLogger = NewSimpleLogger(LevelInfo)
