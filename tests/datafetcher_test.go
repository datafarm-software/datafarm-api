package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/datafarm-software/datafarm-api/datafetcher"
	"github.com/google/go-cmp/cmp"
)

func TestDeviceDataSliceCsvInfo(t *testing.T) {
	tests := map[string]struct {
		wantErr bool
		input   datafetcher.DeviceDataSlice
		want    datafetcher.CsvInfo
	}{

		"single deviceid, single queryfield to csv info": {
			want: datafetcher.CsvInfo{
				Headers: []string{RegisteredQueryField},
				DeviceIdIndexes: map[datafetcher.DeviceId]datafetcher.Indexes{
					RegisteredDeviceId: {0, 1},
				},
			},
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

		"single deviceid, multiple queryfields to csv info": {
			want: datafetcher.CsvInfo{
				Headers: []string{AnotherRegisteredQueryField, RegisteredQueryField},
				DeviceIdIndexes: map[datafetcher.DeviceId]datafetcher.Indexes{
					RegisteredDeviceId: {0, 1},
				},
			},
			input: datafetcher.DeviceDataSlice{
				{
					DeviceID:   RegisteredDeviceId,
					Timestamp:  InsideTimeRange,
					SensorData: map[string]float64{RegisteredQueryField: 24},
				},
				{
					DeviceID:   RegisteredDeviceId,
					Timestamp:  AlsoInsideTimeRange,
					SensorData: map[string]float64{AnotherRegisteredQueryField: 80},
				},
			},
		},

		"multiple deviceids but same queryfields": {
			want: datafetcher.CsvInfo{
				Headers: []string{RegisteredQueryField},
				DeviceIdIndexes: map[datafetcher.DeviceId]datafetcher.Indexes{
					RegisteredDeviceId:        {0},
					AnotherRegisteredDeviceId: {1},
				},
			},
			input: datafetcher.DeviceDataSlice{
				{
					DeviceID:   RegisteredDeviceId,
					Timestamp:  InsideTimeRange,
					SensorData: map[string]float64{RegisteredQueryField: 24},
				},
				{
					DeviceID:   AnotherRegisteredDeviceId,
					Timestamp:  AlsoInsideTimeRange,
					SensorData: map[string]float64{RegisteredQueryField: 25},
				},
			},
		},

		"multiple deviceids multiple queryfields": {
			want: datafetcher.CsvInfo{
				Headers: []string{AnotherRegisteredQueryField, RegisteredQueryField},
				DeviceIdIndexes: map[datafetcher.DeviceId]datafetcher.Indexes{
					RegisteredDeviceId:        {0},
					AnotherRegisteredDeviceId: {1},
				},
			},
			input: datafetcher.DeviceDataSlice{
				{
					DeviceID:   AnotherRegisteredDeviceId,
					Timestamp:  AlsoInsideTimeRange,
					SensorData: map[string]float64{AnotherRegisteredQueryField: 80},
				},
				{
					DeviceID:   RegisteredDeviceId,
					Timestamp:  InsideTimeRange,
					SensorData: map[string]float64{RegisteredQueryField: 24},
				},
			},
		},
	}

	var got datafetcher.CsvInfo
	var err error
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err = tc.input.CsvInfo()
			if tc.wantErr != (err != nil) {
				t.Fatalf("wantErr: %v, err: %v", tc.wantErr, err)
			}
			if !tc.wantErr {
				if diff := cmp.Diff(tc.want, got); diff != "" {
					t.Fatalf("csvinfo mismatch: %v\n", diff)
				}
			}
		})
	}
}

func TestDeviceDataSliceCsv(t *testing.T) {
	tests := map[string]struct {
		wantErr bool
		input   datafetcher.DeviceDataSlice
		want    string
	}{

		"single deviceid, single queryfield to csv": {
			want: fmt.Sprintf(",%s\n%s,\n%s,%s\n%s,%s\n",
				RegisteredQueryField, RegisteredDeviceId,
				InsideTimeRange.Format(time.DateTime), "24.000",
				AlsoInsideTimeRange.Format(time.DateTime), "25.000",
			),
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

		"single deviceid, multiple queryfield to csv": {
			want: fmt.Sprintf(",%s,%s\n%s,,\n%s,,%s\n%s,%s,\n",
				AnotherRegisteredQueryField, RegisteredQueryField, RegisteredDeviceId,
				InsideTimeRange.Format(time.DateTime), "24.000",
				AlsoInsideTimeRange.Format(time.DateTime), "80.000",
			),
			input: datafetcher.DeviceDataSlice{
				{
					DeviceID:   RegisteredDeviceId,
					Timestamp:  InsideTimeRange,
					SensorData: map[string]float64{RegisteredQueryField: 24},
				},
				{
					DeviceID:   RegisteredDeviceId,
					Timestamp:  AlsoInsideTimeRange,
					SensorData: map[string]float64{AnotherRegisteredQueryField: 80},
				},
			},
		},

		"multiple deviceids but same queryfields": {
			want: fmt.Sprintf(",%s\n%s,\n%s,%s\n%s,\n%s,%s\n",
				RegisteredQueryField, RegisteredDeviceId,
				InsideTimeRange.Format(time.DateTime), "24.000",
				AnotherRegisteredDeviceId,
				AlsoInsideTimeRange.Format(time.DateTime), "25.000",
			),
			input: datafetcher.DeviceDataSlice{
				{
					DeviceID:   RegisteredDeviceId,
					Timestamp:  InsideTimeRange,
					SensorData: map[string]float64{RegisteredQueryField: 24},
				},
				{
					DeviceID:   AnotherRegisteredDeviceId,
					Timestamp:  AlsoInsideTimeRange,
					SensorData: map[string]float64{RegisteredQueryField: 25},
				},
			},
		},

		"multiple deviceids multiple queryfields": {
			want: fmt.Sprintf(",%s,%s\n%s,,\n%s,,%s\n%s,,\n%s,%s,\n",
				AnotherRegisteredQueryField, RegisteredQueryField, RegisteredDeviceId,
				InsideTimeRange.Format(time.DateTime), "24.000",
				AnotherRegisteredDeviceId,
				AlsoInsideTimeRange.Format(time.DateTime), "80.000",
			),
			input: datafetcher.DeviceDataSlice{
				{
					DeviceID:   RegisteredDeviceId,
					Timestamp:  InsideTimeRange,
					SensorData: map[string]float64{RegisteredQueryField: 24},
				},
				{
					DeviceID:   AnotherRegisteredDeviceId,
					Timestamp:  AlsoInsideTimeRange,
					SensorData: map[string]float64{AnotherRegisteredQueryField: 80},
				},
			},
		},
	}

	var got string
	var err error
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err = tc.input.Csv()
			if tc.wantErr != (err != nil) {
				t.Fatalf("wantErr: %v, err: %v", tc.wantErr, err)
			}
			if !tc.wantErr {
				if diff := cmp.Diff(tc.want, got); diff != "" {
					t.Fatalf("csv string mismatch: %v\n", diff)
				}
			}
		})
	}
}
