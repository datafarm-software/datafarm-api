package logging

type Logger interface {
	Warn(msg string, metadata map[string]string)
	Error(msg string, metadata map[string]string)
	Info(msg string, metadata map[string]string)
}
