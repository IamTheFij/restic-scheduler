package main_test

import (
	"errors"
	"testing"

	main "git.iamthefij.com/iamthefij/restic-scheduler"
)

func TestResticConfigValidate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		config      main.ResticConfig
		expectedErr error
	}{
		{
			name:        "missing passphrase",
			expectedErr: main.ErrMutuallyExclusive,
			config:      main.ResticConfig{}, // nolint:exhaustivestruct
		},
		{
			name:        "passphrase no file",
			expectedErr: nil,
			// nolint:exhaustivestruct
			config: main.ResticConfig{
				Passphrase: "shh",
			},
		},
		{
			name:        "file no passphrase",
			expectedErr: nil,
			// nolint:exhaustivestruct
			config: main.ResticConfig{
				GlobalOpts: &main.ResticGlobalOpts{
					PasswordFile: "file",
				},
			},
		},
		{
			name:        "file and passphrase",
			expectedErr: main.ErrMutuallyExclusive,
			// nolint:exhaustivestruct
			config: main.ResticConfig{
				Passphrase: "shh",
				GlobalOpts: &main.ResticGlobalOpts{
					PasswordFile: "file",
				},
			},
		},
	}

	for _, c := range cases {
		testCase := c

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			actual := testCase.config.Validate()

			if !errors.Is(actual, testCase.expectedErr) {
				t.Errorf("expected error to wrap %v but found %v", testCase.expectedErr, actual)
			}
		})
	}
}
