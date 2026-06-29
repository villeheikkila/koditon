package telemetry

import "testing"

func TestHTTPSignalEndpointURL(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		endpoint   string
		signalPath string
		want       string
	}{
		"collector base URL": {
			endpoint:   "http://localhost:4318",
			signalPath: "/v1/logs",
			want:       "http://localhost:4318/v1/logs",
		},
		"collector base URL with slash": {
			endpoint:   "http://localhost:4318/",
			signalPath: "/v1/traces",
			want:       "http://localhost:4318/v1/traces",
		},
		"Traceway OTLP base URL": {
			endpoint:   "https://traceway.example.com/api/otel",
			signalPath: "/v1/metrics",
			want:       "https://traceway.example.com/api/otel/v1/metrics",
		},
		"explicit signal URL": {
			endpoint:   "http://localhost:4318/v1/logs",
			signalPath: "/v1/logs",
			want:       "http://localhost:4318/v1/logs",
		},
		"hostport endpoint": {
			endpoint:   "localhost:4318",
			signalPath: "/v1/logs",
			want:       "",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := httpSignalEndpointURL(tt.endpoint, tt.signalPath); got != tt.want {
				t.Fatalf("httpSignalEndpointURL(%q, %q) = %q, want %q", tt.endpoint, tt.signalPath, got, tt.want)
			}
		})
	}
}
