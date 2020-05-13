package errors

import (
	"go.uber.org/zap/zapcore"
)

// AgentError is an error that occurs in the a log agent.
type AgentError struct {
	Description string
	Suggestion  string
	Details     ErrorDetails
	Stack       ErrorStack
}

// Error will return the error message.
func (e AgentError) Error() string {
	return e.Description
}

// MarshalLogObject will define the representation of this error when logging.
func (e AgentError) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddString("description", e.Description)

	if e.Suggestion != "" {
		encoder.AddString("suggestion", e.Suggestion)
	}

	if len(e.Details) != 0 {
		encoder.AddObject("details", e.Details)
	}

	return nil
}

// WithDetails will add details to an agent error.
func WithDetails(err error, keyValues ...string) error {
	if agentErr, ok := err.(AgentError); ok {
		if len(keyValues) > 0 {
			for i := 0; i+1 < len(keyValues); i += 2 {
				agentErr.Details[keyValues[i]] = keyValues[i+1]
			}
		}
		return agentErr
	}
	return NewError(err.Error(), "", keyValues...)
}

// NewError will create a new agent error.
func NewError(description string, suggestion string, keyValues ...string) AgentError {
	return AgentError{
		Description: description,
		Suggestion:  suggestion,
		Details:     createDetails(keyValues),
		Stack:       createStack(),
	}
}
