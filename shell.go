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

type CapturedLogWriter struct {
	Lines  []string
	logger *log.Logger
}

func NewCapturedLogWriter(logger *log.Logger) *CapturedLogWriter {
	return &CapturedLogWriter{Lines: []string{}, logger: logger}
}

func (w *CapturedLogWriter) Write(content []byte) (n int, err error) {
	message := string(content)
	for _, line := range strings.Split(message, "\n") {
		w.Lines = append(w.Lines, line)
		w.logger.Printf(" %s", line)
	}

	return len(content), nil
}

func RunShell(script string, cwd string, env map[string]string, logger *log.Logger) error {
	cmd := exec.Command("sh", "-c", strings.TrimSpace(script)) // nolint:gosec

	// Make both stderr and stdout go to logger
	cmd.Stdout = NewCapturedLogWriter(logger)
	cmd.Stderr = cmd.Stdout

	// Set working directory
	cmd.Dir = cwd

	// Convert env to list if values provided
	if len(env) > 0 {
		envList := os.Environ()
		envList = append(envList, EnvMapToList(env)...)
		cmd.Env = envList
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("shell execution failed: %w", err)
	}

	return nil
}
