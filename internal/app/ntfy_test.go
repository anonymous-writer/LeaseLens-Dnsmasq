package app

import "testing"

func TestValidateNTFYURL(t *testing.T) {
	for _, tc := range []struct {
		name    string
		url     string
		wantErr bool
	}{
		{name: "https topic", url: "https://ntfy.sh/leaselens-test"},
		{name: "http self hosted", url: "http://127.0.0.1:8080/leaselens"},
		{name: "missing scheme", url: "ntfy.sh/topic", wantErr: true},
		{name: "missing host", url: "https:///topic", wantErr: true},
		{name: "unsupported scheme", url: "ftp://example.com/topic", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateNTFYURL(tc.url)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateNTFYURL(%q) error = %v, wantErr %v", tc.url, err, tc.wantErr)
			}
		})
	}
}

func TestValidNTFYPriority(t *testing.T) {
	for input, want := range map[string]string{
		"min": "1", "low": "2", "default": "3", "high": "4", "urgent": "5", "unknown": "4",
	} {
		if got := validNTFYPriority(input); got != want {
			t.Errorf("validNTFYPriority(%q) = %q, want %q", input, got, want)
		}
	}
}
