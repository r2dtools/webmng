package logger

type LoggerInterface interface {
	Error(message string, args ...interface{})
	Warning(message string, args ...interface{})
	Info(message string, args ...interface{})
	Debug(message string, args ...interface{})
}

type NilLogger struct{}

func (l NilLogger) Error(message string, args ...interface{})   {}
func (l NilLogger) Warning(message string, args ...interface{}) {}
func (l NilLogger) Info(message string, args ...interface{})    {}
func (l NilLogger) Debug(message string, args ...interface{})   {}
