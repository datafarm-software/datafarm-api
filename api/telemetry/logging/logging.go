package logging

import (
	"context"
	"reflect"
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

func FromTagMetadata(a any) (m Metadata) {
	m.KeyValue = make(map[string]string)
	t := reflect.TypeOf(a)
	v := reflect.ValueOf(a)
	if !v.IsValid() {
		return
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return
	}
	var field reflect.StructField
	var val reflect.Value
	var key string
	var strSlice []string
	var ok bool
	for i := 0; i < t.NumField(); i++ {
		key = ""
		field = t.Field(i)
		key = field.Tag.Get("log")
		if key == "" {
			continue
		}
		val = v.Field(i)
		switch val.Kind() {
		case reflect.String:
			m.KeyValue[key] = val.String()
		case reflect.Slice:
			strSlice, ok = val.Interface().([]string)
			if !ok {
				continue
			}
			m.KeySlice[key] = strSlice
		default:
			continue
		}
	}
	return
}

type MockLogger struct{}

func (l *MockLogger) Close(context.Context) error         { return nil }
func (l *MockLogger) Warn(msg string, metadata Metadata)  {}
func (l *MockLogger) Error(msg string, metadata Metadata) {}
func (l *MockLogger) Info(msg string, metadata Metadata)  {}
