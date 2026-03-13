package jow

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const (
	baseURL          = "https://api.jow.fr/public"
	availabilityZone = "FR"
)

// Client is a Jow API HTTP client
type Client struct {
	token      string
	httpClient *http.Client
}

// userID extracts the userId claim from the Bearer JWT token.
func (c *Client) userID() (string, error) {
	parts := strings.Split(c.token, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid JWT format")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("decode JWT payload: %w", err)
	}
	var claims struct {
		UserID string `json:"userId"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", fmt.Errorf("parse JWT claims: %w", err)
	}
	if claims.UserID == "" {
		return "", fmt.Errorf("userId not found in JWT")
	}
	return claims.UserID, nil
}

// NewClient returns a new Jow client authenticated with the given Bearer token.
// The token can be retrieved from the browser's developer tools when logged in to jow.fr.
func NewClient(bearerToken string) *Client {
	return &Client{
		token:      bearerToken,
		httpClient: &http.Client{},
	}
}

// do executes an HTTP request and returns the raw response body.
func (c *Client) do(method, path string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	var rawBody []byte
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		rawBody = data
		bodyReader = bytes.NewReader(data)
	}

	if method == "POST" || method == "PUT" {
		fmt.Fprintf(os.Stderr, "[DEBUG] %s %s\nBody: %s\n", method, path, string(rawBody))
	}

	req, err := http.NewRequest(method, baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "fr")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Origin", "https://jow.fr")
	req.Header.Set("Referer", "https://jow.fr/")
	req.Header.Set("x-jow-withmeta", "1")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
