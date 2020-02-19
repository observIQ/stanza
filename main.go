package log

type LogCollector struct {
	Config Config
}

func (l *LogCollector) Start() error {
	return nil
}

func (l *LogCollector) Stop() {}

func (l *LogCollector) Status() struct{} {
	return struct{}{}
}
