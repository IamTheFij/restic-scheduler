package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
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

// Write writes the provided byte slice to the logger and stores each captured line.
func (w *CapturedLogWriter) Write(content []byte) (n int, err error) {
	message := string(content)
	for _, line := range strings.Split(message, "\n") {
		w.Lines = append(w.Lines, line)
		w.logger.Printf(" %s", line)
	}

	return len(content), nil
}

// LinesMergedWith returns a slice of lines from this logger merged with another.
func (w CapturedLogWriter) LinesMergedWith(other CapturedLogWriter) []string {
	allLines := []string{}
	allLines = append(allLines, w.Lines...)
	allLines = append(allLines, other.Lines...)

	sort.Strings(allLines)

	return allLines
}

type CapturedCommandLogWriter struct {
	Stdout *CapturedLogWriter
	Stderr *CapturedLogWriter
}

func NewCapturedCommandLogWriter(logger *log.Logger) *CapturedCommandLogWriter {
	return &CapturedCommandLogWriter{
		Stdout: NewCapturedLogWriter(logger),
		Stderr: NewCapturedLogWriter(logger),
	}
}

func (cclw CapturedCommandLogWriter) AllLines() []string {
	return cclw.Stdout.LinesMergedWith(*cclw.Stderr)
}

func RunShell(script string, cwd string, env map[string]string, logger *log.Logger) error {
	cmd := exec.Command("sh", "-c", strings.TrimSpace(script)) //nolint:gosec

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
