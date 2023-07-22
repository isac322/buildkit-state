package internal

import (
	"testing"

	"github.com/sethvargo/go-githubactions"
	"github.com/stretchr/testify/assert"
)

func Test_getMultilineInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		env          map[string]string
		variableName string
		expected     []string
	}{
		{
			name:         "does not exist",
			env:          nil,
			variableName: "some-name",
			expected:     nil,
		},
		{
			name:         "empty",
			env:          map[string]string{"INPUT_SOME-NAME": ""},
			variableName: "some-name",
			expected:     nil,
		},
		{
			name:         "only empty line",
			env:          map[string]string{"INPUT_SOME-NAME": "\n\n"},
			variableName: "some-name",
			expected:     nil,
		},
		{
			name:         "single",
			env:          map[string]string{"INPUT_SOME-NAME": "single"},
			variableName: "some-name",
			expected:     []string{"single"},
		},
		{
			name:         "multiple without empty line",
			env:          map[string]string{"INPUT_SOME-NAME": "val1\nval2"},
			variableName: "some-name",
			expected:     []string{"val1", "val2"},
		},
		{
			name:         "multiple trailing empty lines",
			env:          map[string]string{"INPUT_SOME-NAME": "val1\nval2\n\n"},
			variableName: "some-name",
			expected:     []string{"val1", "val2"},
		},
		{
			name:         "multiple mixed empty lines",
			env:          map[string]string{"INPUT_SOME-NAME": "val1\n\n\nval2"},
			variableName: "some-name",
			expected:     []string{"val1", "val2"},
		},
		{
			name:         "multiple starts with empty lines",
			env:          map[string]string{"INPUT_SOME-NAME": "\n\nval1\nval2"},
			variableName: "some-name",
			expected:     []string{"val1", "val2"},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gha := githubactions.New(githubactions.WithGetenv(func(key string) string {
				return tc.env[key]
			}))
			actual := getMultilineInput(gha, tc.variableName)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestBuildKitContainerNameFromBuilder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "some_name",
			expected: "buildx_buildkit_some_name0",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()

			actual := BuildKitContainerNameFromBuilder(tc.input)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
