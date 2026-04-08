package client

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
)

// StreamDeploymentLogs opens a server-sent-events connection to the build log
// stream for the given deployment and calls handle for each log chunk.
// handle receives the raw log text (not JSON). When the server signals
// completion the function returns nil.
func (c *Client) StreamDeploymentLogs(ctx context.Context, jobID, deploymentID string, handle func(chunk string) error) error {
	fullURL, err := url.Parse(c.baseURL)
	if err != nil {
		return err
	}
	fullURL.Path = path.Join(fullURL.Path, "/v1/jobs", jobID, "deployments", deploymentID, "logs")
	q := fullURL.Query()
	q.Set("stream", "true")
	fullURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.streamHTTP.Do(req)
	if err != nil {
		return err
	}

	var closeOnce sync.Once
	closeBody := func() { _ = resp.Body.Close() }

	done := make(chan struct{})
	defer func() {
		close(done)
		closeOnce.Do(closeBody)
	}()

	go func() {
		select {
		case <-ctx.Done():
			closeOnce.Do(closeBody)
		case <-done:
		}
	}()

	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		var apiErr map[string]any
		if jsonErr := json.Unmarshal(body, &apiErr); jsonErr == nil {
			if msg, ok := apiErr["error"].(string); ok && msg != "" {
				return fmt.Errorf("deployment log stream failed (%d): %s", resp.StatusCode, msg)
			}
		}
		return fmt.Errorf("deployment log stream failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var dataLines []string

	flush := func() error {
		if len(dataLines) == 0 {
			return nil
		}
		payload := strings.Join(dataLines, "\n")
		dataLines = nil

		var chunk DeploymentLogChunk
		if jsonErr := json.Unmarshal([]byte(payload), &chunk); jsonErr != nil {
			// Not JSON — pass raw text through.
			return handle(payload)
		}
		if chunk.Done {
			return io.EOF
		}
		if chunk.Chunk != "" {
			return handle(chunk.Chunk)
		}
		return nil
	}

	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case line == "":
			if err := flush(); err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				return err
			}
		case strings.HasPrefix(line, ":"):
			continue
		case strings.HasPrefix(line, "event:"):
			continue
		case strings.HasPrefix(line, "data:"):
			dataLines = append(dataLines, trimSSEData(line))
		}
	}

	if err := scanner.Err(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}
	if err := flush(); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return ctx.Err()
}

func trimSSEData(line string) string {
	v := strings.TrimPrefix(line, "data:")
	if strings.HasPrefix(v, " ") {
		return v[1:]
	}
	return v
}
