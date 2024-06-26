package log

type NoOpLogger struct {
}

func (l *NoOpLogger) Debug(message string, fields ...Field) {}
func (l *NoOpLogger) Info(message string, fields ...Field)  {}
func (l *NoOpLogger) Warn(message string, fields ...Field)  {}
func (l *NoOpLogger) Error(message string, fields ...Field) {}
