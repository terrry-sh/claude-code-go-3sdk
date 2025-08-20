package claudesdk

import (
	"fmt"
)

// CLIError is the base error type for CLI-related errors
type CLIError struct {
	Message string
	Cause   error
}

func (e *CLIError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *CLIError) Unwrap() error {
	return e.Cause
}

// CLINotFoundError indicates the Claude CLI was not found
type CLINotFoundError struct {
	CLIError
}

// CLIConnectionError indicates a connection error with the CLI
type CLIConnectionError struct {
	CLIError
}

// ProcessError indicates the CLI process failed
type ProcessError struct {
	CLIError
	ExitCode int
	Stderr   string
}

func (e *ProcessError) Error() string {
	msg := fmt.Sprintf("Command failed with exit code %d", e.ExitCode)
	if e.Stderr != "" {
		msg += fmt.Sprintf("\nStderr: %s", e.Stderr)
	}
	return msg
}

// MessageParseError indicates an error parsing a message
type MessageParseError struct {
	CLIError
	Data interface{}
}

func (e *MessageParseError) Error() string {
	return fmt.Sprintf("Failed to parse message: %s", e.Message)
}

// JSONDecodeError indicates an error decoding JSON
type JSONDecodeError struct {
	CLIError
}

// NewCLINotFoundError creates a new CLINotFoundError
func NewCLINotFoundError(message string) *CLINotFoundError {
	return &CLINotFoundError{
		CLIError: CLIError{Message: message},
	}
}

// NewCLIConnectionError creates a new CLIConnectionError
func NewCLIConnectionError(message string) *CLIConnectionError {
	return &CLIConnectionError{
		CLIError: CLIError{Message: message},
	}
}

// NewProcessError creates a new ProcessError
func NewProcessError(message string, exitCode int, stderr string) *ProcessError {
	return &ProcessError{
		CLIError: CLIError{Message: message},
		ExitCode: exitCode,
		Stderr:   stderr,
	}
}

// NewMessageParseError creates a new MessageParseError
func NewMessageParseError(message string, data interface{}) *MessageParseError {
	return &MessageParseError{
		CLIError: CLIError{Message: message},
		Data:     data,
	}
}

// NewJSONDecodeError creates a new JSONDecodeError
func NewJSONDecodeError(message string, cause error) *JSONDecodeError {
	return &JSONDecodeError{
		CLIError: CLIError{Message: message, Cause: cause},
	}
}