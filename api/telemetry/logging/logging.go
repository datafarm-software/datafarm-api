package logging

import (
	"context"
	"reflect"

	"github.com/fatih/structtag"
	"github.com/mitchellh/reflectwalk"
)

type Logger interface {
	Close(context.Context) error
	Warn(msg string, metadata Metadata)
	Error(msg string, metadata Metadata)
	Info(msg string, metadata Metadata)
}

type Metadata struct {
	KeyValue map[string][]string
}

type metadataWalker struct {
	metadata Metadata
	tags     []string
}

func (w *metadataWalker) StructField(
	field reflect.StructField,
	value reflect.Value,
) error {
	tags, err := structtag.Parse(string(field.Tag))
	if err != nil {
		return err
	}
	tag, err := tags.Get("log")
	if err != nil {
		w.tags = append(w.tags, "")
		return nil
	}
	w.tags = append(w.tags, tag.Name)
	return nil
}

func (w *metadataWalker) Exit(loc reflectwalk.Location) error {
	if loc == reflectwalk.StructField {
		w.tags = w.tags[:len(w.tags)-1]
	}
	return nil
}

func (w *metadataWalker) Primitive(v reflect.Value) error {
	if len(w.tags) == 0 {
		return nil
	}
	tag := w.tags[len(w.tags)-1]
	if tag == "" || v.Kind() != reflect.String {
		return nil
	}
	w.metadata.KeyValue[tag] =
		append(w.metadata.KeyValue[tag], v.String())
	return nil
}

func FromTagMetadata(a any) (m Metadata, err error) {
	w := &metadataWalker{
		metadata: Metadata{
			KeyValue: make(map[string][]string),
		},
	}
	err = reflectwalk.Walk(a, w)
	return w.metadata, err
}

type MockLogger struct{}

func (l *MockLogger) Close(context.Context) error         { return nil }
func (l *MockLogger) Warn(msg string, metadata Metadata)  {}
func (l *MockLogger) Error(msg string, metadata Metadata) {}
func (l *MockLogger) Info(msg string, metadata Metadata)  {}
