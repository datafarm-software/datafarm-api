package tests

import (
	"testing"

	"github.com/datafarm-software/datafarm-api/datafetcher"
	"github.com/google/go-cmp/cmp"
)

func TestDeviceDataSliceCsvHeaders(t *testing.T) {
	tests := map[string]struct {
		wantErr bool
		input   datafetcher.DeviceDataSlice
		want    []string
	}{
		"successfully convert a device data slice queryfields to csv headers": {
			want: []string{
				RegisteredDeviceId, RegisteredQueryField, AnotherRegisteredQueryField},
			input: datafetcher.DeviceDataSlice{
				{
					DeviceID:   RegisteredDeviceId,
					Timestamp:  InsideTimeRange,
					SensorData: map[string]float64{RegisteredQueryField: 24},
				},
				{
					DeviceID:   RegisteredDeviceId,
					Timestamp:  AlsoInsideTimeRange,
					SensorData: map[string]float64{RegisteredQueryField: 25},
				},
			},
		},
	}
	var got []string
	var err error
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err = tc.input.CsvHeaders()
			if tc.wantErr != (err != nil) {
				t.Fatalf("wantErr: %v, err: %v", tc.wantErr, err)
			}
			if !tc.wantErr {
				if diff := cmp.Diff(tc.want, got); diff != "" {
					t.Fatalf("headers mismatch: %v\n", diff)
				}
			}
		})
	}
}
