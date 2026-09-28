package logger

import (
	"fmt"
	"io"
	"strings"
)

// initError общее сообщение starter, которое дублирует ошибку компонента
const initError = "application initialization error"

// Logger это логгер starter для консольного инструмента
// Без verbose служебные сообщения starter скрываются, ошибки печатаются всегда
type Logger struct {
	w       io.Writer
	verbose bool
}

// New возвращает логгер
func New(w io.Writer, verbose bool) *Logger {
	return &Logger{w: w, verbose: verbose}
}

// Println печатает сообщение
func (l *Logger) Println(v ...any) {
	l.print(strings.TrimSuffix(fmt.Sprintln(v...), "\n"))
}

// Printf печатает форматированное сообщение
func (l *Logger) Printf(format string, v ...any) {
	l.print(fmt.Sprintf(format, v...))
}

func (l *Logger) print(msg string) {
	if !l.verbose && isNoise(msg) {
		return
	}
	fmt.Fprintln(l.w, msg)
}

// isNoise сообщает, является ли сообщение служебным выводом starter
func isNoise(msg string) bool {
	switch {
	case msg == initError, msg == "shutdown ...":
		return true
	case strings.HasSuffix(msg, " is OK"):
		return true
	case strings.HasPrefix(msg, "Graceful shutdown"), strings.HasPrefix(msg, "  - "):
		return true
	case strings.HasPrefix(msg, "service ") && strings.HasSuffix(msg, " is stopped"):
		return true
	}
	return false
}
