package domain_test

import (
	"strings"
	"testing"

	"latencyops/internal/domain"
)

func TestValidateURL_SSRFProtection(t *testing.T) {
	tests := []struct {
		name      string
		targetURL string
		expectErr bool
		errPrefix string
	}{
		{
			name:      "Valid Public HTTPS URL",
			targetURL: "https://api.stripe.com/v1/health",
			expectErr: false,
		},
		{
			name:      "Valid Public HTTP URL",
			targetURL: "http://example.com",
			expectErr: false,
		},
		{
			name:      "SSRF Attempt - Localhost",
			targetURL: "http://127.0.0.1:8080/admin",
			expectErr: true,
			errPrefix: "SSRF protection triggered",
		},
		{
			name:      "SSRF Attempt - Internal 10.x.x.x Network",
			targetURL: "https://10.0.1.55/metrics",
			expectErr: true,
			errPrefix: "SSRF protection triggered",
		},
		{
			name:      "SSRF Attempt - Cloud Metadata (AWS)",
			targetURL: "http://169.254.169.254/latest/meta-data/",
			expectErr: true,
			errPrefix: "SSRF protection triggered",
		},
		{
			name:      "Invalid Scheme - FTP",
			targetURL: "ftp://files.example.com",
			expectErr: true,
			errPrefix: "invalid scheme",
		},
		{
			name:      "Unresolvable Host",
			targetURL: "https://this-domain-definitely-does-not-exist.local",
			expectErr: true,
			errPrefix: "failed to resolve hostname",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := domain.ValidateURL(tt.targetURL)
			
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error for URL %s, but got none", tt.targetURL)
				}
				if !strings.HasPrefix(err.Error(), tt.errPrefix) {
					t.Errorf("expected error to start with %q, got %q", tt.errPrefix, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("did not expect error for URL %s, got: %v", tt.targetURL, err)
				}
			}
		})
	}
}