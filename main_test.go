package main

import (
	"reflect"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected map[string]string
	}{
		{
			name:     "empty args",
			args:     []string{},
			expected: map[string]string{},
		},
		{
			name:     "single valid arg",
			args:     []string{"thanos=v1.0.0"},
			expected: map[string]string{"thanos": "v1.0.0"},
		},
		{
			name:     "multiple valid args",
			args:     []string{"thanos1=v1.0.0", "thanos2=main", "thanos3=a1b2c3d"},
			expected: map[string]string{"thanos1": "v1.0.0", "thanos2": "main", "thanos3": "a1b2c3d"},
		},
		{
			name:     "mixed valid and invalid args",
			args:     []string{"thanos1=v1.0.0", "invalid", "thanos2=main"},
			expected: map[string]string{"thanos1": "v1.0.0", "thanos2": "main"},
		},
		{
			name:     "arg with equals in value",
			args:     []string{"thanos=branch=feature"},
			expected: map[string]string{"thanos": "branch=feature"},
		},
		{
			name:     "arg with path and ref",
			args:     []string{"path/to/thanos=v2.1.0"},
			expected: map[string]string{"path/to/thanos": "v2.1.0"},
		},
		{
			name:     "all invalid args",
			args:     []string{"invalid1", "invalid2", "invalid3"},
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseArgs(tt.args)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parseArgs(%v) = %v, want %v", tt.args, result, tt.expected)
			}
		})
	}
}

func TestRunCommandVerbose(t *testing.T) {
	// Test verbose mode
	originalVerbose := verbose
	defer func() { verbose = originalVerbose }()

	verbose = true
	err := runCommand(".", "echo", "verbose test")
	if err != nil {
		t.Errorf("runCommand() in verbose mode failed: %v", err)
	}

	verbose = false
	err = runCommand(".", "echo", "non-verbose test")
	if err != nil {
		t.Errorf("runCommand() in non-verbose mode failed: %v", err)
	}
}
