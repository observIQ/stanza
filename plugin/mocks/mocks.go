package mocks

func NewMockPlugin(id string) *Plugin {
	mockOutput := &Plugin{}
	mockOutput.On("ID").Return(id)
	mockOutput.On("CanProcess").Return(true)
	mockOutput.On("CanOutput").Return(true)
	return mockOutput
}
