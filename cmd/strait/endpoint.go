package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/strait-dev/cli/internal/client"
	"github.com/strait-dev/cli/internal/styles"
	"github.com/strait-dev/cli/internal/wizard"
	"github.com/strait-dev/strait-go/serve"

	"github.com/spf13/cobra"
)

// endpointVerifyTransport is the HTTP client used by `endpoint verify`. Tests
// swap this with an httptest server's client.
var endpointVerifyTransport = &http.Client{Timeout: 30 * time.Second}

func newEndpointCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "endpoint",
		Short: "Manage HTTPS push endpoints for SDK-defined jobs",
		Long: `Manage the endpoint URLs that the Strait scheduler pushes signed
payloads to. Each job has an endpoint_url that points at the customer's
serve handler (strait.serve from the SDK).`,
	}
	cmd.AddCommand(newEndpointSetCommand(state))
	cmd.AddCommand(newEndpointGetCommand(state))
	cmd.AddCommand(newEndpointVerifyCommand(state))
	return cmd
}

func newEndpointSetCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <job-slug> <url>",
		Short: "Set the endpoint URL for a job",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := strings.TrimSpace(args[0])
			endpoint := strings.TrimSpace(args[1])
			if err := wizard.ValidateEndpoint(endpoint); err != nil {
				return err
			}
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			id, err := resolveJobIdentifier(cmd.Context(), cli, state, slug)
			if err != nil {
				return err
			}
			updated, err := cli.UpdateJob(cmd.Context(), id, client.UpdateJobRequest{EndpointURL: &endpoint})
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Endpoint updated for "+styles.Bold.Render(updated.Slug)))
				fmt.Fprintln(os.Stderr, styles.KeyValue("URL", updated.EndpointURL))
				return nil
			}
			return printData(state, updated)
		},
	}
	return cmd
}

func newEndpointGetCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <job-slug>",
		Short: "Print the endpoint URL for a job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			id, err := resolveJobIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			job, err := cli.GetJob(cmd.Context(), id)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.KeyValue("Job", job.Slug))
				fmt.Fprintln(os.Stderr, styles.KeyValue("Endpoint", job.EndpointURL))
				if job.FallbackEndpointURL != "" {
					fmt.Fprintln(os.Stderr, styles.KeyValue("Fallback", job.FallbackEndpointURL))
				}
				return nil
			}
			return printData(state, map[string]any{
				"slug":                  job.Slug,
				"endpoint_url":          job.EndpointURL,
				"fallback_endpoint_url": job.FallbackEndpointURL,
			})
		},
	}
	return cmd
}

func newEndpointVerifyCommand(state *appState) *cobra.Command {
	var secret string
	cmd := &cobra.Command{
		Use:   "verify <job-slug>",
		Short: "Send a signed canary payload to the registered endpoint",
		Long: `Send a signed canary payload to the job's registered endpoint URL
and report whether the handler responded correctly.

The signing secret must be supplied via --secret or the STRAIT_SIGNING_SECRET
environment variable. Signatures are produced by github.com/strait-dev/strait-go/serve
("v1=<hex>" HMAC-SHA256 over "<timestamp>.<body>") so any endpoint built with
the official serve adapter will accept the canary.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			signingSecret := strings.TrimSpace(secret)
			if signingSecret == "" {
				signingSecret = strings.TrimSpace(os.Getenv("STRAIT_SIGNING_SECRET"))
			}
			if signingSecret == "" {
				return fmt.Errorf("signing secret is required: pass --secret or set STRAIT_SIGNING_SECRET")
			}

			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			id, err := resolveJobIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}
			job, err := cli.GetJob(cmd.Context(), id)
			if err != nil {
				return err
			}
			if strings.TrimSpace(job.EndpointURL) == "" {
				return fmt.Errorf("job %q has no endpoint_url; use `strait endpoint set` first", job.Slug)
			}

			result, verr := postCanary(cmd.Context(), endpointVerifyTransport, job.EndpointURL, job.Slug, signingSecret)
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.KeyValue("Endpoint", job.EndpointURL))
				fmt.Fprintln(os.Stderr, styles.KeyValue("Status", result.Status))
				fmt.Fprintln(os.Stderr, styles.KeyValue("HTTP", fmt.Sprintf("%d", result.HTTPStatus)))
				if result.Error != "" {
					fmt.Fprintln(os.Stderr, styles.Warn(result.Error))
				}
				if verr != nil {
					return verr
				}
				if result.Status != "verified" {
					return fmt.Errorf("endpoint verification failed (status=%q)", result.Status)
				}
				fmt.Fprintln(os.Stderr, styles.Success("Endpoint verified"))
				return nil
			}
			if perr := printData(state, result); perr != nil {
				return perr
			}
			if verr != nil {
				return verr
			}
			if result.Status != "verified" {
				return fmt.Errorf("endpoint verification failed (status=%q)", result.Status)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&secret, "secret", "", "HMAC signing secret (or set STRAIT_SIGNING_SECRET)")
	return cmd
}

// verifyCanarySlug is sent as the canary's job_slug. A correctly-wired
// serve handler returns 404 ("no handler for job slug") which proves the
// HMAC layer accepted the request without running any user job code.
const verifyCanarySlug = "__strait_verify__"

// verifyResult is the parsed outcome of a canary POST.
type verifyResult struct {
	Status     string `json:"status"`
	HTTPStatus int    `json:"http_status"`
	Error      string `json:"error,omitempty"`
	RunID      string `json:"run_id,omitempty"`
}

// postCanary signs and POSTs a canary payload to endpointURL using the same
// HMAC scheme as github.com/strait-dev/strait-go/serve. The canary uses the
// reserved slug "__strait_verify__" so a correctly-wired endpoint reaches
// the registry lookup (HMAC verified) and returns 404 without running any
// user job code. That 404 path is what we report as "verified".
//
// Outcomes:
//   - 404 ("no handler for job slug") + jobSlug echoed in error: verified
//   - 401 ("signature verification failed"): bad secret
//   - 2xx: a handler was registered for the canary slug and ran successfully
//     (also reported as verified, since the HMAC layer accepted the request)
//   - Anything else: failed, with the body surfaced in result.Error.
//
// A non-nil error indicates a transport-level failure (build, dial, read).
func postCanary(ctx context.Context, hc *http.Client, endpointURL, jobSlug, secret string) (verifyResult, error) {
	runID, err := newVerifyRunID()
	if err != nil {
		return verifyResult{Status: "failed", Error: err.Error()}, fmt.Errorf("generate run id: %w", err)
	}
	body, err := json.Marshal(map[string]any{
		"job_slug": verifyCanarySlug,
		"run_id":   runID,
		"payload":  map[string]any{"__verify": true, "for_slug": jobSlug},
	})
	if err != nil {
		return verifyResult{Status: "failed", Error: err.Error()}, fmt.Errorf("marshal payload: %w", err)
	}

	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	signature := serve.Sign([]byte(secret), timestamp, body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpointURL, bytes.NewReader(body))
	if err != nil {
		return verifyResult{Status: "failed", Error: err.Error(), RunID: runID}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Strait-Signature", signature)
	req.Header.Set("X-Strait-Timestamp", timestamp)

	resp, err := hc.Do(req)
	if err != nil {
		return verifyResult{Status: "failed", Error: err.Error(), RunID: runID}, fmt.Errorf("post: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	result := verifyResult{HTTPStatus: resp.StatusCode, RunID: runID}

	switch {
	case resp.StatusCode == http.StatusNotFound:
		// HMAC verified, no handler registered for the canary slug — exactly
		// what we want.
		result.Status = "verified"
		return result, nil
	case resp.StatusCode == http.StatusUnauthorized:
		result.Status = "failed"
		result.Error = "HMAC verification failed at the receiver (check signing secret)"
		return result, nil
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		// A handler was registered for the canary slug. Treat any successful
		// dispatch as verified — the user opted into accepting the canary.
		var parsed struct {
			Success bool   `json:"success"`
			Error   string `json:"error,omitempty"`
		}
		if err := json.Unmarshal(respBody, &parsed); err != nil {
			result.Status = "failed"
			result.Error = fmt.Sprintf("malformed response: %v", err)
			return result, nil
		}
		if !parsed.Success {
			result.Status = "failed"
			result.Error = parsed.Error
			return result, nil
		}
		result.Status = "verified"
		return result, nil
	default:
		result.Status = "failed"
		result.Error = strings.TrimSpace(string(respBody))
		return result, nil
	}
}

func newVerifyRunID() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "verify-" + hex.EncodeToString(buf), nil
}
