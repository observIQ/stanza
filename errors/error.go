package errors

import (
	"encoding/json"
	"fmt"

	"go.uber.org/zap/zapcore"
)

// AgentError is an error that occurs in the agent.
type AgentError struct {
	Description string
	Suggestion  string
	Details     ErrorDetails
}

// Error will return the error message.
func (e AgentError) Error() string {
	if len(e.Details) == 0 {
		return e.Description
	}
	marshalled, _ := json.Marshal(e.Details)
	return fmt.Sprintf("%s: %s", e.Description, string(marshalled))
}

// MarshalLogObject will define the representation of this error when logging.
func (e AgentError) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddString("description", e.Description)

	if e.Suggestion != "" {
		encoder.AddString("suggestion", e.Suggestion)
	}

	if len(e.Details) != 0 {
		_ = encoder.AddObject("details", e.Details)
	}

	return nil
}

// WithDetails will return the error with additional details
func (e AgentError) WithDetails(keyValues ...string) AgentError {
	return WithDetails(e, keyValues...)
}

// WithDetails will add details to an agent error
func WithDetails(err error, keyValues ...string) AgentError {
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

// Wrap adds context to the description for richer logs
func Wrap(err error, context string) AgentError {
	if agentErr, ok := err.(AgentError); ok {
		agentErr.Description = fmt.Sprintf("%s: %s", context, agentErr.Description)
		return agentErr
	}

	return NewError(fmt.Sprintf("%s: %s", context, err.Error()), "")
}

// NewError will create a new agent error.
func NewError(description string, suggestion string, keyValues ...string) AgentError {
	return AgentError{
		Description: description,
		Suggestion:  suggestion,
		Details:     createDetails(keyValues),
	}
}
