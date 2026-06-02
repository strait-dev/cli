// Package client provides an HTTP client for the Strait REST API.
// It handles authentication, JSON encoding/decoding, retry with exponential
// backoff for transient failures, and structured error responses.
package client

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
)

// Client is an HTTP client for the Strait REST API.
type Client struct {
	baseURL    string
	apiKey     string
	http       *http.Client
	streamHTTP *http.Client
}

// New creates a new API client.
func New(baseURL, apiKey string, timeout time.Duration) (*Client, error) {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmed == "" {
		return nil, fmt.Errorf("base URL is required")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("base URL must be http(s)")
	}

	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	// The stream client uses no overall timeout (SSE connections are long-lived)
	// but limits the TLS handshake and response-header phase to prevent hangs
	// when the server accepts TCP but never responds.
	streamTransport := http.DefaultTransport.(*http.Transport).Clone()
	streamTransport.TLSHandshakeTimeout = 10 * time.Second
	streamTransport.ResponseHeaderTimeout = 30 * time.Second

	return &Client{
		baseURL:    parsed.String(),
		apiKey:     strings.TrimSpace(apiKey),
		http:       &http.Client{Timeout: timeout},
		streamHTTP: &http.Client{Timeout: 0, Transport: streamTransport},
	}, nil
}

// SetTransport replaces the transport used by the non-streaming HTTP client.
// This is primarily used to inject a debug-logging transport.
func (c *Client) SetTransport(rt http.RoundTripper) {
	c.http.Transport = rt
}

func (c *Client) doJSON(ctx context.Context, method, endpoint string, query url.Values, body any, out any) error {
	return c.doJSONWithHeaders(ctx, method, endpoint, query, body, nil, out)
}

// paginatedResponse wraps the paginated API envelope.
type paginatedResponse struct {
	Data       json.RawMessage `json:"data"`
	NextCursor *string         `json:"next_cursor,omitempty"`
	HasMore    bool            `json:"has_more"`
}

// doListJSON performs a GET request and decodes a list response into out.
//
// The Strait API is inconsistent: some list endpoints return a paginated
// envelope ({"data":[...],"has_more":...}) while others return a bare JSON
// array ([...]). This helper accepts either shape so list commands work
// against every endpoint regardless of which form the server uses.
func (c *Client) doListJSON(ctx context.Context, endpoint string, query url.Values, out any) error {
	var raw json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, endpoint, query, nil, &raw); err != nil {
		return err
	}
	return unmarshalListBody(raw, out)
}

// unmarshalListBody decodes either a bare JSON array or a paginated envelope's
// data field into out. An absent/empty envelope data field leaves out at its
// zero value (an empty slice).
func unmarshalListBody(raw json.RawMessage, out any) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		return json.Unmarshal(trimmed, out)
	}
	var envelope paginatedResponse
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return err
	}
	if len(envelope.Data) == 0 {
		return nil
	}
	return json.Unmarshal(envelope.Data, out)
}

// doListAllJSON auto-paginates a list endpoint by following next_cursor.
func (c *Client) doListAllJSON(ctx context.Context, endpoint string, query url.Values, out any) error {
	const maxPages = 100

	q := url.Values{}
	maps.Copy(q, query)
	q.Set("limit", "100")

	var allData []json.RawMessage
	var pages int
	for range maxPages {
		var raw json.RawMessage
		if err := c.doJSON(ctx, http.MethodGet, endpoint, q, nil, &raw); err != nil {
			return err
		}

		// A bare array response is unpaginated: collect it and stop.
		if trimmed := bytes.TrimSpace(raw); len(trimmed) > 0 && trimmed[0] == '[' {
			var items []json.RawMessage
			if err := json.Unmarshal(trimmed, &items); err != nil {
				return fmt.Errorf("decode list data: %w", err)
			}
			allData = append(allData, items...)
			pages++
			break
		}

		var envelope paginatedResponse
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return err
		}

		if len(envelope.Data) > 0 {
			var items []json.RawMessage
			if err := json.Unmarshal(envelope.Data, &items); err != nil {
				return fmt.Errorf("decode paginated data: %w", err)
			}
			allData = append(allData, items...)
		}
		pages++

		if !envelope.HasMore || envelope.NextCursor == nil {
			break
		}
		q.Set("cursor", *envelope.NextCursor)
	}

	if pages >= maxPages {
		fmt.Fprintf(os.Stderr, "warning: results truncated at %d items (pagination limit reached)\n", len(allData))
	}

	merged, err := json.Marshal(allData)
	if err != nil {
		return fmt.Errorf("merge paginated data: %w", err)
	}
	return json.Unmarshal(merged, out)
}

// parseErrorBody extracts a human-readable message and machine-readable code
// from a non-2xx response body. The Strait API returns errors as
// {"error":{"code":"...","message":"..."}} (object form); older/edge responses
// may use {"error":"..."} (string form). Falls back to the trimmed raw body
// when nothing structured is found.
func parseErrorBody(body []byte) (message, code string) {
	message = strings.TrimSpace(string(body))

	var envelope struct {
		Error json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil || len(envelope.Error) == 0 {
		return message, ""
	}

	var obj struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(envelope.Error, &obj); err == nil && (obj.Message != "" || obj.Code != "") {
		if obj.Message != "" {
			message = obj.Message
		}
		return message, obj.Code
	}

	var s string
	if err := json.Unmarshal(envelope.Error, &s); err == nil && s != "" {
		return s, ""
	}

	return message, ""
}

func (c *Client) doJSONWithHeaders(ctx context.Context, method, endpoint string, query url.Values, body any, headers map[string]string, out any) error {
	fullURL, err := url.Parse(c.baseURL)
	if err != nil {
		return err
	}
	fullURL.Path = path.Join(fullURL.Path, endpoint)
	if query != nil {
		fullURL.RawQuery = query.Encode()
	}

	var bodyBytes []byte
	if body != nil {
		var marshalErr error
		bodyBytes, marshalErr = json.Marshal(body)
		if marshalErr != nil {
			return marshalErr
		}
	}

	const maxRetries = 3
	var lastErr error

	for attempt := range maxRetries {
		var bodyReader io.Reader
		if bodyBytes != nil {
			bodyReader = bytes.NewReader(bodyBytes)
		}

		req, reqErr := http.NewRequestWithContext(ctx, method, fullURL.String(), bodyReader)
		if reqErr != nil {
			return reqErr
		}
		req.Header.Set("Accept", "application/json")
		if bodyBytes != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if c.apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+c.apiKey)
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, doErr := c.http.Do(req)
		if doErr != nil {
			return doErr
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError {
			_ = resp.Body.Close()
			lastErr = &APIError{StatusCode: resp.StatusCode, Message: http.StatusText(resp.StatusCode), Op: "request"}
			if attempt < maxRetries-1 {
				backoff := time.Duration(1<<uint(attempt)) * time.Second
				backoff += jitter(backoff / 4) // add up to 25% jitter
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(backoff):
				}
			}
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode >= http.StatusBadRequest {
			errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			msg, code := parseErrorBody(errBody)
			return &APIError{StatusCode: resp.StatusCode, Message: msg, Code: code, Op: "request"}
		}

		// 204 No Content (and any empty-body success) has nothing to decode.
		// Decoding an empty body returns io.EOF, which previously surfaced as a
		// spurious "EOF" error on every delete/update/action endpoint.
		if out == nil || resp.StatusCode == http.StatusNoContent {
			return nil
		}
		// Cap decoded response to 50 MB to prevent unbounded memory allocation
		// from a malicious or misconfigured server.
		const maxResponseBody = 50 * 1024 * 1024
		return json.NewDecoder(io.LimitReader(resp.Body, maxResponseBody)).Decode(out)
	}

	return lastErr
}

// UploadFile performs an HTTP PUT of r to the given (presigned) URL.
// size must be the exact byte count that will be read from r.
// No authorization header is added; the URL is expected to be self-authenticating (presigned).
func (c *Client) UploadFile(ctx context.Context, uploadURL string, r io.Reader, size int64) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, r)
	if err != nil {
		return fmt.Errorf("create upload request: %w", err)
	}
	req.ContentLength = size
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("upload: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return &APIError{StatusCode: resp.StatusCode, Message: strings.TrimSpace(string(body)), Op: "upload"}
	}
	return nil
}

// RawRequest sends an authenticated HTTP request to the given path (relative to
// the base URL) with an optional JSON body. The response body is written to w
// as indented JSON when parseable, otherwise as-is. This is used by the
// `debug request` subcommand for interactive API inspection.
func (c *Client) RawRequest(ctx context.Context, method, urlPath string, body string, w io.Writer) error {
	fullURL := strings.TrimRight(c.baseURL, "/") + "/" + strings.TrimLeft(urlPath, "/")

	var bodyReader io.Reader
	if strings.TrimSpace(body) != "" {
		bodyReader = bytes.NewBufferString(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if bodyReader != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	const maxBody = 10 * 1024 * 1024
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	// Print status line to stderr so stdout stays clean for piping.
	fmt.Fprintf(os.Stderr, "HTTP %d\n", resp.StatusCode)

	// Pretty-print JSON if parseable; otherwise raw.
	var pretty bytes.Buffer
	if json.Indent(&pretty, respBody, "", "  ") == nil {
		_, err = io.Copy(w, &pretty)
	} else {
		_, err = w.Write(respBody)
	}
	if err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	// Ensure trailing newline.
	if len(respBody) > 0 && respBody[len(respBody)-1] != '\n' {
		_, _ = fmt.Fprintln(w)
	}
	return nil
}

// jitter returns a random duration in [0, maxJitter) using crypto/rand.
func jitter(maxJitter time.Duration) time.Duration {
	if maxJitter <= 0 {
		return 0
	}
	var buf [8]byte
	_, _ = rand.Read(buf[:])
	n := binary.LittleEndian.Uint64(buf[:])
	return time.Duration(n % uint64(maxJitter)) //nolint:gosec // jitter overflow is harmless — wraps to a valid positive duration
}
