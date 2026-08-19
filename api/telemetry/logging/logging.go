package logging

import "context"

type Logger interface {
	Close(context.Context) error
	Warn(msg string, metadata map[string]string)
	Error(msg string, metadata map[string]string)
	Info(msg string, metadata map[string]string)
}

type MockLogger struct{}

func (l *MockLogger) Close(context.Context) error { return nil }
func (l *MockLogger) Warn(msg string, metadata map[string]string)
func (l *MockLogger) Error(msg string, metadata map[string]string)
func (l *MockLogger) Info(msg string, metadata map[string]string)
