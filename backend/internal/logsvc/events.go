package logsvc

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
)

func Info(location string, message string) {
	log.Printf("INFO 位置：%s。%s", cleanLocation(location), sanitizeLine(message))
}

func Error(location string, err error) {
	if err == nil {
		return
	}
	ErrorMessage(location, err.Error())
}

func ErrorMessage(location string, message string) {
	log.Printf("ERROR 位置：%s。错误原因：%s", cleanLocation(location), sanitizeLine(message))
}

func Panic(location string, value any) {
	log.Printf("PANIC 位置：%s。错误原因：%v\n%s", cleanLocation(location), value, debug.Stack())
}

func Recover(location string) {
	if value := recover(); value != nil {
		Panic(location, value)
	}
}

func RecoverAndExit(location string) {
	if value := recover(); value != nil {
		Panic(location, value)
		CloseDefault()
		os.Exit(2)
	}
}

func Caller(skip int) string {
	pc, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return "unknown"
	}
	fn := runtime.FuncForPC(pc)
	name := "unknown"
	if fn != nil {
		name = fn.Name()
	}
	return fmt.Sprintf("%s (%s:%d)", name, filepath.ToSlash(file), line)
}

func cleanLocation(location string) string {
	location = strings.TrimSpace(location)
	if location == "" {
		return "unknown"
	}
	return sanitizeLine(location)
}

func sanitizeLine(value string) string {
	value = strings.ReplaceAll(value, "\r", "\\r")
	value = strings.ReplaceAll(value, "\n", "\\n")
	return strings.TrimSpace(value)
}
