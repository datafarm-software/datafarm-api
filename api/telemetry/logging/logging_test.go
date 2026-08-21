package logging

import (
	"testing"

	"github.com/datafarm-software/datafarm-api/api/datafetcher"
	deviceinfo "github.com/datafarm-software/datafarm-api/api/device-info"
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
					"deviceid":    {"123"},
					"queryfields": {"1", "2", "3"},
					"timezone":    {"Africa/Johannesburg"},
					"start":       {"-2d"},
					"stop":        {"now"},
				},
			},
		},

		"parse pointer to sensordatarequest": {
			input: &datafetcher.SensorDataRequest{
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
					"deviceid":    {"123"},
					"queryfields": {"1", "2", "3"},
					"timezone":    {"Africa/Johannesburg"},
					"start":       {"-2d"},
					"stop":        {"now"},
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
					"deviceid":    {"123"},
					"queryfields": {"1", "2", "3"},
					"start":       {"-2d"},
				},
			},
		},

		"dont parse struct without log tags": {
			input: struct{ Some string }{"Some"},
			want: Metadata{
				KeyValue: map[string][]string{},
			},
		},

		"dont parse pointer to struct without log tags": {
			input: struct{ Some string }{"Some"},
			want: Metadata{
				KeyValue: map[string][]string{},
			},
		},

		"parse batch sensordatarequest": {
			input: datafetcher.BatchSensorDataRequest{
				Hardware: []datafetcher.Hardware{
					{
						DeviceId:    "1",
						QueryFields: []string{"1", "2", "3"},
					},
					{
						DeviceId:    "2",
						QueryFields: []string{"1"},
					},
				},
				TimeFrame: datafetcher.TimeFrame{
					Start: "-2d",
				},
			},
			want: Metadata{
				KeyValue: map[string][]string{
					"deviceid":    {"1", "2"},
					"queryfields": {"1", "2", "3", "1"},
					"start":       {"-2d"},
				},
			},
		},

		"parse pointer to batch sensordatarequest": {
			input: &datafetcher.BatchSensorDataRequest{
				Hardware: []datafetcher.Hardware{
					{
						DeviceId:    "1",
						QueryFields: []string{"1", "2", "3"},
					},
					{
						DeviceId:    "2",
						QueryFields: []string{"1"},
					},
				},
				TimeFrame: datafetcher.TimeFrame{
					Start: "-2d",
				},
			},
			want: Metadata{
				KeyValue: map[string][]string{
					"deviceid":    {"1", "2"},
					"queryfields": {"1", "2", "3", "1"},
					"start":       {"-2d"},
				},
			},
		},

		"parse batch queryfieldsrequest": {
			input: deviceinfo.BatchQueryFieldsRequest{
				Body: deviceinfo.DeviceBatch{
					DeviceIds: []string{"1", "2"},
				},
			},
			want: Metadata{
				KeyValue: map[string][]string{
					"deviceids": {"1", "2"},
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
