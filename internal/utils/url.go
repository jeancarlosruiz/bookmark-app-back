package utils

import (
	"net/url"
	"strings"
)

// NormalizeURL normalizes a URL to ensure consistent comparison
// This helps prevent duplicates like:
// - https://example.com vs https://example.com/
// - https://example.com?b=2&a=1 vs https://example.com?a=1&b=2
func NormalizeURL(rawURL string) (string, error) {
	// Parse the URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	// Ensure scheme is lowercase
	parsedURL.Scheme = strings.ToLower(parsedURL.Scheme)

	// Ensure host is lowercase (domains are case-insensitive)
	parsedURL.Host = strings.ToLower(parsedURL.Host)

	// Remove trailing slash from path unless it's the root path
	if parsedURL.Path != "/" && strings.HasSuffix(parsedURL.Path, "/") {
		parsedURL.Path = strings.TrimSuffix(parsedURL.Path, "/")
	}

	// Ensure root path is exactly "/" if no path is specified
	if parsedURL.Path == "" {
		parsedURL.Path = "/"
	}

	// Sort query parameters for consistent ordering
	query := parsedURL.Query()
	parsedURL.RawQuery = query.Encode()

	// Return the normalized URL
	return parsedURL.String(), nil
}
