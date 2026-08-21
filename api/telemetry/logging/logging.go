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
}

func (w *metadataWalker) Struct(reflect.Value) error { return nil }

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
		return nil
	}
	value = reflect.Indirect(value)
	switch value.Kind() {
	case reflect.String:
		w.metadata.KeyValue[tag.Name] =
			append(w.metadata.KeyValue[tag.Name], value.String())
	case reflect.Slice:
		for i := range value.Len() {
			elem := value.Index(i)
			if elem.Kind() != reflect.String {
				continue
			}
			w.metadata.KeyValue[tag.Name] =
				append(w.metadata.KeyValue[tag.Name], elem.String())
		}
	}
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
