package logging

import "context"

type Logger interface {
	Close(context.Context) error
	Warn(msg string, metadata map[string]string)
	Error(msg string, metadata map[string]string)
	Info(msg string, metadata map[string]string)
}
