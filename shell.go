package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

var (
	loggerFlags = log.LstdFlags | log.Lmsgprefix
	loggers     = map[string]*log.Logger{}
)

func GetLogger(name string) *log.Logger {
	if logger, ok := loggers[name]; ok {
		return logger
	}

	logger := log.New(os.Stderr, name+":", loggerFlags)
	loggers[name] = logger

	return logger
}

func GetChildLogger(parent *log.Logger, name string) *log.Logger {
	childName := fmt.Sprintf("%s%s", parent.Prefix(), name)

	return GetLogger(childName)
}

type logWriter struct {
	logger *log.Logger
}

func NewLogWriter(logger *log.Logger) *logWriter {
	return &logWriter{logger}
}

func (w logWriter) Write(p []byte) (n int, err error) {
	message := fmt.Sprintf("%s", p)
	for _, line := range strings.Split(message, "\n") {
		w.logger.Printf(" %s", line)
	}

	return len(p), nil
}

func RunShell(script string, cwd string, env map[string]string, logger *log.Logger) error {
	cmd := exec.Command("sh", "-c", strings.TrimSpace(script))

	// Make both stderr and stdout go to logger
	// fmt.Println("LOGGER PREFIX", logger.Prefix())
	// logger.Println("From logger")
	cmd.Stdout = NewLogWriter(logger)
	cmd.Stderr = cmd.Stdout

	// Set working directory
	cmd.Dir = cwd

	// Convert env to list if values provided
	if len(env) > 0 {
		envList := os.Environ()

		for name, value := range env {
			envList = append(envList, fmt.Sprintf("%s=%s", name, value))
		}

		cmd.Env = envList
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("shell execution failed: %w", err)
	}

	return nil
}
