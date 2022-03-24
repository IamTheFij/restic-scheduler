package main_test

import (
	"testing"

	main "git.iamthefij.com/iamthefij/restic-scheduler"
	"github.com/go-test/deep"
)

func AssertEqual(t *testing.T, message string, expected, actual interface{}) bool {
	t.Helper()

	if diff := deep.Equal(expected, actual); diff != nil {
		t.Errorf("%s: %v", message, diff)

		return false
	}

	return true
}

func AssertEqualFail(t *testing.T, message string, expected, actual interface{}) {
	t.Helper()

	if !AssertEqual(t, message, expected, actual) {
		t.FailNow()
	}
}

func TestMergeEnvMap(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		parent   map[string]string
		child    map[string]string
		expected map[string]string
	}{
		{
			name: "No child",
			parent: map[string]string{
				"key": "value",
			},
			child: nil,
			expected: map[string]string{
				"key": "value",
			},
		},
		{
			name:   "No parent",
			parent: nil,
			child: map[string]string{
				"key": "value",
			},
			expected: map[string]string{
				"key": "value",
			},
		},
		{
			name: "Overwrite value",
			parent: map[string]string{
				"key":   "old",
				"other": "other",
			},
			child: map[string]string{
				"key": "new",
			},
			expected: map[string]string{
				"key":   "new",
				"other": "other",
			},
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			actual := main.MergeEnvMap(c.parent, c.child)
			if diff := deep.Equal(c.expected, actual); diff != nil {
				t.Error(diff)
			}
		})
	}
}

func TestEnvMapToList(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"key": "value",
	}
	expected := []string{
		"key=value",
	}
	actual := main.EnvMapToList(env)

	if diff := deep.Equal(expected, actual); diff != nil {
		t.Error(diff)
	}
}
