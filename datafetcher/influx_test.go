package datafetcher

import "testing"

func TestApi_formatQueryRange(t *testing.T) {
	tests := []struct {
		name      string
		startTime string
		stopTime  string
		want      string
		wantErr   bool
	}{
		{
			name:      "relative time",
			startTime: "-6d",
			stopTime:  "",
			want:      "start: -6d",
			wantErr:   false,
		},
		{
			name:      "no start time provided",
			startTime: "",
			stopTime:  "",
			want:      "",
			wantErr:   true,
		},
		{
			name:      "rfc3339 start time provided, no stop time",
			startTime: "2025-01-29T08:00:00Z",
			stopTime:  "",
			want:      "",
			wantErr:   true,
		},
		{
			name:      "rfc3339 start and stop time provided",
			startTime: "2025-01-28T08:00:00Z",
			stopTime:  "2025-01-29T08:00:00Z",
			want:      "start: 2025-01-28T08:00:00Z, stop: 2025-01-29T08:00:00Z",
			wantErr:   false,
		},
		{
			name:      "relative start time with rfc3339 stop time",
			startTime: "-6d",
			stopTime:  "2025-01-29T08:00:00Z",
			want:      "start: -6d",
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := InfluxDatafetcher{}
			got, gotErr := i.formatQueryRange(tt.startTime, tt.stopTime)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("formatQueryRange() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("formatQueryRange() succeeded unexpectedly")
			}
			if got != tt.want {
				t.Errorf("formatQueryRange() = %s, want %s", got, tt.want)
			}
		})
	}
}
