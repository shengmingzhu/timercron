package timerlog

import (
	"os"
)

// Logger does underlying logging work for timerlog.
type Logger interface {
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
}

var logger Logger

// SetupLogger directs log output to l.
func SetupLogger(l Logger) {
	logger = l
}

// Info logs to the INFO log.
func Info(args ...interface{}) {
	if logger == nil {
		return
	}
	logger.Info(args...)
}

// Infof logs to the INFO log. Arguments are handled in the manner of fmt.Printf.
func Infof(format string, args ...interface{}) {
	if logger == nil {
		return
	}
	logger.Infof(format, args...)
}

// Warn logs to the WARNING log.
func Warn(args ...interface{}) {
	if logger == nil {
		return
	}
	logger.Warn(args...)
}

// Warnf logs to the WARNING log. Arguments are handled in the manner of fmt.Printf.
func Warnf(format string, args ...interface{}) {
	if logger == nil {
		return
	}
	logger.Warnf(format, args...)
}

// Error logs to the ERROR log.
func Error(args ...interface{}) {
	if logger == nil {
		return
	}
	logger.Error(args...)
}

// Errorf logs to the ERROR log. Arguments are handled in the manner of fmt.Printf.
func Errorf(format string, args ...interface{}) {
	if logger == nil {
		return
	}
	logger.Errorf(format, args...)
}

// Fatal logs to the FATAL log. Arguments are handled in the manner of fmt.Print.
// It calls os.Exit() with exit code 1.
func Fatal(args ...interface{}) {
	if logger == nil {
		return
	}
	logger.Fatal(args...)
	// Make sure fatal logs will exit.
	os.Exit(1)
}

// Fatalf logs to the FATAL log. Arguments are handled in the manner of fmt.Printf.
// It calles os.Exit() with exit code 1.
func Fatalf(format string, args ...interface{}) {
	if logger == nil {
		return
	}
	logger.Fatalf(format, args...)
	// Make sure fatal logs will exit.
	os.Exit(1)
}
