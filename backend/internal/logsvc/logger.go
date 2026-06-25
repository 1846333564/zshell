package logsvc

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	logFolderName = "log"
	logRetention  = 24 * time.Hour
)

type Logger struct {
	dir         string
	now         func() time.Time
	mu          sync.Mutex
	currentHour string
	file        *os.File
}

var defaultLogger *Logger

func InitDefault() (*Logger, error) {
	dir, err := DefaultLogDir()
	if err != nil {
		return nil, err
	}
	logger, err := New(dir)
	if err != nil {
		return nil, err
	}
	log.SetOutput(logger)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	defaultLogger = logger
	log.Printf("日志系统已启动，目录：%s", dir)
	return logger, nil
}

func DefaultLogDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(configDir, "zShell", logFolderName), nil
}

func New(dir string) (*Logger, error) {
	return newWithClock(dir, time.Now)
}

func newWithClock(dir string, now func() time.Time) (*Logger, error) {
	if now == nil {
		now = time.Now
	}
	logger := &Logger{dir: dir, now: now}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}
	if err := logger.cleanupOldLogs(); err != nil {
		return nil, err
	}
	return logger, nil
}

func (l *Logger) Write(p []byte) (int, error) {
	if l == nil {
		return len(p), nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.rotateLocked(l.now()); err != nil {
		return 0, err
	}
	return l.file.Write(p)
}

func (l *Logger) Close() error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.closeLocked()
}

func (l *Logger) Dir() string {
	if l == nil {
		return ""
	}
	return l.dir
}

func CloseDefault() {
	if defaultLogger != nil {
		_ = defaultLogger.Close()
	}
}

func (l *Logger) rotateLocked(now time.Time) error {
	hour := now.Format("20060102-15")
	if l.file != nil && l.currentHour == hour {
		return nil
	}
	if err := l.closeLocked(); err != nil {
		return err
	}

	path := filepath.Join(l.dir, "zshell-"+hour+".log")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open hourly log file: %w", err)
	}
	l.file = file
	l.currentHour = hour
	return nil
}

func (l *Logger) closeLocked() error {
	if l.file == nil {
		return nil
	}
	err := l.file.Close()
	l.file = nil
	l.currentHour = ""
	return err
}

func (l *Logger) cleanupOldLogs() error {
	entries, err := os.ReadDir(l.dir)
	if err != nil {
		return fmt.Errorf("read log dir: %w", err)
	}
	cutoff := l.now().Add(-logRetention)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if !strings.HasPrefix(entry.Name(), "zshell-") || !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(l.dir, entry.Name()))
		}
	}
	return nil
}
