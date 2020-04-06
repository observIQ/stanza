package helper

// BasicLifecycle provies a basic implementation of a plugin lifecycle.
type BasicLifecycle struct {
	Running bool
}

// Start will set the lifecycle to a running state.
func (s *BasicLifecycle) Start() error {
	s.Running = true
	return nil
}

// Stop will set the lifecycle to a stopped state.
func (s *BasicLifecycle) Stop() error {
	s.Running = false
	return nil
}
