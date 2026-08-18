package domain

import (
	"fmt"
	"net"
	"net/url"
	"time"
)

// Endpoint represents a user-defined API target for our worker engine.
type Endpoint struct {
	ID          string    // UUIDv4 or KSUID to prevent enumeration
	WorkspaceID string    // Mandatory for tenant isolation
	Name        string
	TargetURL   string
	Interval    int       // Check frequency in seconds (e.g., 60)
	CreatedAt   time.Time
}

// PingResult represents the output from our concurrent worker pool.
type PingResult struct {
	WorkspaceID string        `json:"workspace_id"`
	EndpointID  string        `json:"endpoint_id"`
	StatusCode  int           `json:"status_code"`
	LatencyMs   int64         `json:"latency_ms"`
	Latency     time.Duration `json:"-"` // Internal use only; LatencyMs is the serialized form
	IsUp        bool          `json:"is_up"`
	Timestamp   time.Time     `json:"timestamp"`
}

// AlertRule defines the threshold criteria for triggering webhooks.
type AlertRule struct {
	ID               string
	WorkspaceID      string
	EndpointID       string
	MaxLatencyMs     int
	AllowedErrorRate float64 // e.g., trigger if > 0%
}

// privateCIDRs defines the blocked subnets to mitigate SSRF (OWASP API7).
var privateCIDRs []*net.IPNet

func init() {
	// Initialize the blocked ranges as defined in our security guidelines.
	ranges := []string{
		"127.0.0.0/8",
		"10.0.0.0/8",
		"192.168.0.0/16",
		"169.254.0.0/16", // Blocks AWS/GCP metadata endpoints
	}

	for _, cidr := range ranges {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Errorf("failed to parse hardcoded CIDR %s: %w", cidr, err))
		}
		privateCIDRs = append(privateCIDRs, ipNet)
	}
}

// ValidateURL parses the target URL, resolves its IP, and ensures it doesn't hit internal networks.
func ValidateURL(rawURL string) error {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("failed to parse url: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("invalid scheme: %s (only http/https allowed)", parsedURL.Scheme)
	}

	hostname := parsedURL.Hostname()
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return fmt.Errorf("failed to resolve hostname %s: %w", hostname, err)
	}

	for _, ip := range ips {
		for _, cidr := range privateCIDRs {
			if cidr.Contains(ip) {
				return fmt.Errorf("SSRF protection triggered: IP %s falls within blocked internal range %s", ip.String(), cidr.String())
			}
		}
	}

	return nil
}