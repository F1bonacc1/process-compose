package util

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"
)

func TestSetTerminalTitle(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	title := "test-title"
	SetTerminalTitle(title)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	expected := "\033]0;process-compose: test-title\007"
	if output != expected {
		t.Errorf("SetTerminalTitle() = %q, want %q", output, expected)
	}
}

type mockProjectNamer struct {
	name string
	err  error
}

func (m *mockProjectNamer) GetProjectName() (string, error) {
	return m.name, m.err
}

func TestSetProjectNameAsTerminalTitle(t *testing.T) {
	tests := []struct {
		name           string
		mockName       string
		mockErr        error
		expectedOutput string
	}{
		{
			name:           "Success",
			mockName:       "my-project",
			mockErr:        nil,
			expectedOutput: "\033]0;process-compose: my-project\007",
		},
		{
			name:           "Error",
			mockName:       "",
			mockErr:        errors.New("some error"),
			expectedOutput: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			mock := &mockProjectNamer{
				name: tt.mockName,
				err:  tt.mockErr,
			}
			SetProjectNameAsTerminalTitle(mock)

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			if output != tt.expectedOutput {
				t.Errorf("SetProjectNameAsTerminalTitle() output = %q, want %q", output, tt.expectedOutput)
			}
		})
	}
}
