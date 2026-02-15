package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
)

var (
	loggerFlags = log.LstdFlags | log.Lmsgprefix
	loggers     = loggerMap{}
)

// / loggerMap is a type that allows concurrent access to loggers
// / using a sync.Map.
type loggerMap struct {
	m sync.Map
}

// store a logger in the sync map
func (s *loggerMap) store(name string, logger *log.Logger) {
	s.m.Store(name, logger)
}

// retrieve a logger from the sync map by name
func (s *loggerMap) load(name string) (*log.Logger, bool) {
	x, ok := s.m.Load(name)
	if !ok || x == nil {
		return nil, false
	}

	return x.(*log.Logger), true
}

// GetLogger gets a logger by name or creates one if it doesn't exist yet.
func GetLogger(name string) *log.Logger {
	if logger, ok := loggers.load(name); ok {
		return logger
	}

	logger := log.New(os.Stderr, name+":", loggerFlags)
	loggers.store(name, logger)

	return logger
}

// GetChildLogger gets a logger appending the name to the parent logger name.
func GetChildLogger(parent *log.Logger, name string) *log.Logger {
	childName := fmt.Sprintf("%s%s", parent.Prefix(), name)

	return GetLogger(childName)
}

// CapturedLogWriter is a writer that stores the written lines in an array.
type CapturedLogWriter struct {
	Lines  []string
	logger *log.Logger
}

// NewCapturedLogWriter creates a new CapturedLogWriter instance.
func NewCapturedLogWriter(logger *log.Logger) *CapturedLogWriter {
	return &CapturedLogWriter{Lines: []string{}, logger: logger}
}

// Write writes the provided byte slice to the logger and stores each captured line.
func (w *CapturedLogWriter) Write(content []byte) (n int, err error) {
	message := string(content)
	for line := range strings.SplitSeq(message, "\n") {
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

// CapturedCommandLogWriter houses CapturedLogWriter instances for stderr and stdout.
type CapturedCommandLogWriter struct {
	Stdout *CapturedLogWriter
	Stderr *CapturedLogWriter
}

// NewCapturedCommandLogWriter creates a new instance of NewCapturedCommandLogWriter wrapping the provided logger.
func NewCapturedCommandLogWriter(logger *log.Logger) *CapturedCommandLogWriter {
	return &CapturedCommandLogWriter{
		Stdout: NewCapturedLogWriter(logger),
		Stderr: NewCapturedLogWriter(logger),
	}
}

// AllLines returns merged output from the log writers.
func (cclw CapturedCommandLogWriter) AllLines() []string {
	return cclw.Stdout.LinesMergedWith(*cclw.Stderr)
}

// RunShell runs a given script string  in a given directory with the provided environment variables and logs to the provided logger.
func RunShell(script string, cwd string, env map[string]string, logger *log.Logger) error {
	cmd := exec.Command("sh", "-c", strings.TrimSpace(script)) //nolint:gosec

	// Make both stderr and stdout go to logger
	output := NewCapturedCommandLogWriter(logger)
	cmd.Stdout = output.Stdout
	cmd.Stderr = output.Stderr

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
