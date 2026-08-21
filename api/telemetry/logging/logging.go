package logging

import (
	"context"
)

type Logger interface {
	Close(context.Context) error
	Warn(msg string, metadata Metadata)
	Error(msg string, metadata Metadata)
	Info(msg string, metadata Metadata)
}

type Metadata struct {
	KeyValue map[string]string
	KeySlice map[string][]string
}

type RequestMetadataProvider interface {
	Metadata() Metadata
}

type MockLogger struct{}

func (l *MockLogger) Close(context.Context) error         { return nil }
func (l *MockLogger) Warn(msg string, metadata Metadata)  {}
func (l *MockLogger) Error(msg string, metadata Metadata) {}
func (l *MockLogger) Info(msg string, metadata Metadata)  {}
