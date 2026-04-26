package cli

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestRootCommand(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		expectOutput string
		expectError  bool
	}{
		{
			name:         "help flag",
			args:         []string{"--help"},
			expectOutput: "Usage:",
			expectError:  false,
		},
		{
			name:         "version flag",
			args:         []string{"--version"},
			expectOutput: "",
			expectError:  false,
		},
		{
			name:         "no args shows help",
			args:         []string{},
			expectOutput: "Usage:",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout and stderr
			oldStdout := os.Stdout
			oldStderr := os.Stderr
			rOut, wOut, _ := os.Pipe()
			rErr, wErr, _ := os.Pipe()
			os.Stdout = wOut
			os.Stderr = wErr

			// Create command
			cmd := rootCmd
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			wOut.Close()
			wErr.Close()
			os.Stdout = oldStdout
			os.Stderr = oldStderr

			// Read combined output
			var buf bytes.Buffer
			io.Copy(&buf, rOut)
			io.Copy(&buf, rErr)
			output := buf.String()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tt.expectOutput != "" && !strings.Contains(output, tt.expectOutput) {
				t.Errorf("Expected output to contain %q, got %q", tt.expectOutput, output)
			}
		})
	}
}

func TestVersionConstant(t *testing.T) {
	if version != "0.1.0" {
		t.Errorf("Expected version 0.1.0, got %q", version)
	}
}
