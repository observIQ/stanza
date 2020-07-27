package testutil

// NewMockOperator will return a basic operator mock
func NewMockOperator(id string) *Operator {
	mockOutput := &Operator{}
	mockOutput.On("ID").Return(id)
	mockOutput.On("CanProcess").Return(true)
	mockOutput.On("CanOutput").Return(true)
	return mockOutput
}
