package logging

import (
	"testing"

	"github.com/datafarm-software/datafarm-api/api/datafetcher"
	"github.com/google/go-cmp/cmp"
)

func TestFromTagMetadata(t *testing.T) {
	tests := map[string]struct {
		input   any
		want    Metadata
		wantErr bool
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
					"deviceid":   {"123"},
					"queryfield": {"1", "2", "3"},
					"timezone":   {"Africa/Johannesburg"},
					"start":      {"-2d"},
					"stop":       {"now"},
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
					"deviceid":   {"123"},
					"queryfield": {"1", "2", "3"},
					"start":      {"-2d"},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := FromTagMetadata(tc.input)
			if tc.wantErr != (err != nil) {
				t.Fatalf("wantErr: %v, err: %v\n", tc.wantErr, err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("mismatch: (-want +got): %s\n", diff)
			}
		})
	}
}
