package logging

import (
	"testing"

	"github.com/datafarm-software/datafarm-api/api/datafetcher"
	"github.com/google/go-cmp/cmp"
)

func TestFromTagMetadata(t *testing.T) {
	tests := map[string]struct {
		input any
		want  Metadata
	}{

		"parse sensordatarequest": {
			input: datafetcher.SensorDataRequest{
				Hardware: datafetcher.Hardware{
					DeviceId:    "123",
					QueryFields: []string{"1", "2", "3"},
				},
				TimeFrame: datafetcher.TimeFrame{
					Timezone: datafetcher.Timezone{Timezone: "Africa/Johannesburg"},
					Start:    "-2d",
					Stop:     "now",
				},
			},
			want: Metadata{
				KeyValue: map[string][]string{
					"deviceid":   []string{"123"},
					"queryfield": []string{"1", "2", "3"},
					"timezone":   []string{"Africa/Johannesburg"},
					"start":      []string{"-2d"},
					"stop":       []string{"now"},
				},
			},
		},

		"parse sensordatarequest with empty fields": {
			input: datafetcher.SensorDataRequest{
				Hardware: datafetcher.Hardware{
					DeviceId:    "123",
					QueryFields: []string{"1", "2", "3"},
				},
				TimeFrame: datafetcher.TimeFrame{
					Start: "-2d",
				},
			},
			want: Metadata{
				KeyValue: map[string][]string{
					"deviceid":   []string{"123"},
					"queryfield": []string{"1", "2", "3"},
					"start":      []string{"-2d"},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := FromTagMetadata(tc.input)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("mismatch: (-want +got): %s\n", diff)
			}
		})
	}
}
