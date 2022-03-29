package main_test

import (
	"bytes"
	"log"
	"testing"

	main "git.iamthefij.com/iamthefij/restic-scheduler"
)

/*
 * type TestCase interface {
 * 	Run(*testing.T)
 * 	Name() string
 * }
 *
 * type TestCases []TestCase
 *
 * func (c TestCases) Run(t *testing.T) {
 * 	t.Helper()
 *
 * 	for _, tc := range c {
 * 		testCase := tc
 *
 * 		t.Parallel()
 *
 * 		t.Run(tc.Name(), tc.Run(t))
 * 	}
 * }
 */

func TestGetLogger(t *testing.T) {
	t.Parallel()

	initialLogger := main.GetLogger("test")

	t.Run("initial logger", func(t *testing.T) {
		t.Parallel()
		AssertEqual(t, "incorrect logger prefix", "test:", initialLogger.Prefix())
	})

	dupeLogger := main.GetLogger("test")

	t.Run("dupe logger", func(t *testing.T) {
		t.Parallel()
		AssertEqual(t, "incorrect logger prefix", "test:", dupeLogger.Prefix())

		if initialLogger != dupeLogger {
			t.Error("expected reused instance")
		}
	})

	secondLogger := main.GetLogger("test2")

	t.Run("dupe logger", func(t *testing.T) {
		t.Parallel()
		AssertEqual(t, "incorrect logger prefix", "test2:", secondLogger.Prefix())

		if initialLogger == secondLogger {
			t.Error("expected new instance")
		}
	})
}

func TestGetChildLogger(t *testing.T) {
	t.Parallel()

	parentLogger := main.GetLogger("parent")
	childLogger := main.GetChildLogger(parentLogger, "child")

	AssertEqual(t, "unexpected child logger prefix", "parent:child:", childLogger.Prefix())
}

func TestCapturedLogWriter(t *testing.T) {
	t.Parallel()

	buffer := bytes.Buffer{}
	logger := log.New(&buffer, "test:", log.Lmsgprefix)
	capturedLogWriter := main.NewCapturedLogWriter(logger)

	if _, err := capturedLogWriter.Write([]byte("testing")); err != nil {
		t.Fatalf("failed to write to captured log writter: %v", err)
	}

	AssertEqual(t, "buffer contains incorrect values", "test: testing\n", buffer.String())
	AssertEqual(t, "lines contains incorrect values", []string{"testing"}, capturedLogWriter.Lines)
}

func TestRunShell(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		script         string
		cwd            string
		env            map[string]string
		expectedOutput string
		expectedErr    bool
	}{
		{
			name:   "successful script",
			script: "echo $FOO",
			cwd:    ".",
			env: map[string]string{
				"FOO": "bar",
			},
			expectedOutput: "prefix: bar\nprefix: \n",
			expectedErr:    false,
		},
		{
			name:   "failed script",
			script: "echo $FOO\nexit 1",
			cwd:    ".",
			env: map[string]string{
				"FOO": "bar",
			},
			expectedOutput: "prefix: bar\nprefix: \n",
			expectedErr:    true,
		},
	}

	for _, c := range cases {
		testCase := c

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			buffer := bytes.Buffer{}
			logger := log.New(&buffer, "prefix:", log.Lmsgprefix)

			err := main.RunShell(
				testCase.script,
				testCase.cwd,
				testCase.env,
				logger,
			)

			if testCase.expectedErr && err == nil {
				t.Error("expected an error but didn't get one")
			}

			if !testCase.expectedErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			AssertEqual(t, "unexpected output", testCase.expectedOutput, buffer.String())
		})
	}
}
