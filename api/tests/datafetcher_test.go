package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/datafarm-software/datafarm-api/api/sensor"
	"github.com/datafarm-software/datafarm-api/api/sensor/data"
	"github.com/google/go-cmp/cmp"
)

func TestSensorDataSliceCsvInfo(t *testing.T) {
	tests := map[string]struct {
		wantErr bool
		input   data.SensorDataSlice
		want    data.CsvInfo
	}{

		"single deviceid, single queryfield to csv info": {
			want: data.CsvInfo{
				Headers: []string{RegisteredQueryField},
				DeviceIdIndexes: map[sensor.DeviceId]data.Indexes{
					RegisteredDeviceId: {0, 1},
				},
				DeviceIds: []sensor.DeviceId{RegisteredDeviceId},
			},
			input: data.SensorDataSlice{
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
			want: data.CsvInfo{
				Headers: []string{AnotherRegisteredQueryField, RegisteredQueryField},
				DeviceIdIndexes: map[sensor.DeviceId]data.Indexes{
					RegisteredDeviceId: {0, 1},
				},
				DeviceIds: []sensor.DeviceId{RegisteredDeviceId},
			},
			input: data.SensorDataSlice{
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
			want: data.CsvInfo{
				Headers: []string{RegisteredQueryField},
				DeviceIdIndexes: map[sensor.DeviceId]data.Indexes{
					RegisteredDeviceId:        {0},
					AnotherRegisteredDeviceId: {1},
				},
				DeviceIds: []sensor.DeviceId{RegisteredDeviceId, AnotherRegisteredDeviceId},
			},
			input: data.SensorDataSlice{
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
			want: data.CsvInfo{
				Headers: []string{AnotherRegisteredQueryField, RegisteredQueryField},
				DeviceIdIndexes: map[sensor.DeviceId]data.Indexes{
					RegisteredDeviceId:        {0},
					AnotherRegisteredDeviceId: {1},
				},
				DeviceIds: []sensor.DeviceId{RegisteredDeviceId, AnotherRegisteredDeviceId},
			},
			input: data.SensorDataSlice{
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

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := tc.input.CsvInfo()
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

func TestSensorDataSliceCsv(t *testing.T) {
	tests := map[string]struct {
		wantErr bool
		input   data.SensorDataSlice
		want    string
	}{

		"single deviceid, single queryfield to csv": {
			want: fmt.Sprintf(",%s\n%s\n%s,%s\n%s,%s\n",
				RegisteredQueryField, RegisteredDeviceId,
				InsideTimeRange.Format(time.RFC3339), "24.000",
				AlsoInsideTimeRange.Format(time.RFC3339), "25.000",
			),
			input: data.SensorDataSlice{
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
			want: fmt.Sprintf(",%s,%s\n%s\n%s,,%s\n%s,%s,\n",
				AnotherRegisteredQueryField, RegisteredQueryField, RegisteredDeviceId,
				InsideTimeRange.Format(time.RFC3339), "24.000",
				AlsoInsideTimeRange.Format(time.RFC3339), "80.000",
			),
			input: data.SensorDataSlice{
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
			want: fmt.Sprintf(",%s\n%s\n%s,%s\n%s\n%s,%s\n",
				RegisteredQueryField, RegisteredDeviceId,
				InsideTimeRange.Format(time.RFC3339), "24.000",
				AnotherRegisteredDeviceId,
				AlsoInsideTimeRange.Format(time.RFC3339), "25.000",
			),
			input: data.SensorDataSlice{
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
			want: fmt.Sprintf(",%s,%s\n%s\n%s,,%s\n%s\n%s,%s,\n",
				AnotherRegisteredQueryField, RegisteredQueryField, RegisteredDeviceId,
				InsideTimeRange.Format(time.RFC3339), "24.000",
				AnotherRegisteredDeviceId,
				AlsoInsideTimeRange.Format(time.RFC3339), "80.000",
			),
			input: data.SensorDataSlice{
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

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := tc.input.Csv()
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
