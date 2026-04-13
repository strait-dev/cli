package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/spf13/cobra"
	cliconfig "github.com/strait-dev/cli/internal/config"
)

func forceAuthIsTerminal(t *testing.T, fn func(int) bool) {
	t.Helper()
	prev := authIsTerminal
	authIsTerminal = fn
	t.Cleanup(func() {
		authIsTerminal = prev
	})
}

func forceAuthReadSecret(t *testing.T, fn func(int) ([]byte, error)) {
	t.Helper()
	prev := authReadSecret
	authReadSecret = fn
	t.Cleanup(func() {
		authReadSecret = prev
	})
}

func forceAuthValidateAPIKey(t *testing.T, fn func(context.Context, string, string, time.Duration) error) {
	t.Helper()
	prev := authValidateAPIKey
	authValidateAPIKey = func(ctx context.Context, serverURL, apiKey string, timeout time.Duration) error {
		return fn(ctx, serverURL, apiKey, timeout)
	}
	t.Cleanup(func() {
		authValidateAPIKey = prev
	})
}

func forceAuthSaveAPIKey(t *testing.T, fn func(string, string) error) {
	t.Helper()
	prev := authSaveAPIKey
	authSaveAPIKey = fn
	t.Cleanup(func() {
		authSaveAPIKey = prev
	})
}

func forceAuthLoadAPIKey(t *testing.T, fn func(string) (string, error)) {
	t.Helper()
	prev := authLoadAPIKey
	authLoadAPIKey = fn
	t.Cleanup(func() {
		authLoadAPIKey = prev
	})
}

func forceAuthDeleteAPIKey(t *testing.T, fn func(string) error) {
	t.Helper()
	prev := authDeleteAPIKey
	authDeleteAPIKey = fn
	t.Cleanup(func() {
		authDeleteAPIKey = prev
	})
}

func forceAuthLoginWithAPIKey(t *testing.T, fn func(*cobra.Command, *appState, string, bool, string, string) error) {
	t.Helper()
	prev := authLoginWithAPIKey
	authLoginWithAPIKey = fn
	t.Cleanup(func() {
		authLoginWithAPIKey = prev
	})
}

func forceAuthLoginWithDeviceCode(t *testing.T, fn func(*cobra.Command, *appState, string, string) error) {
	t.Helper()
	prev := authLoginWithDeviceCode
	authLoginWithDeviceCode = fn
	t.Cleanup(func() {
		authLoginWithDeviceCode = prev
	})
}

func testConfigPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "config.yaml")
}

func loadSavedConfig(t *testing.T, path string) *cliconfig.File {
	t.Helper()
	loaded, err := cliconfig.Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	return loaded.Data
}

func newAuthTestCommand(t *testing.T) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.SetContext(t.Context())
	return cmd
}

func withClosedStdin(t *testing.T, fn func()) {
	t.Helper()
	orig := os.Stdin
	r, _, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	_ = r.Close()
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = orig
	})
	fn()
}

func TestLoginCommand_UsesTokenMode(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{serverURL: "https://api.default.example", timeout: time.Second}}

	var got struct {
		apiKey        string
		withToken     bool
		targetContext string
		targetServer  string
	}
	forceAuthLoginWithAPIKey(t, func(_ *cobra.Command, _ *appState, apiKey string, withToken bool, targetContext, targetServer string) error {
		got.apiKey = apiKey
		got.withToken = withToken
		got.targetContext = targetContext
		got.targetServer = targetServer
		return nil
	})

	cmd := newLoginCommand(state)
	cmd.SetArgs([]string{"--token", "sk-token", "--context", "prod", "--server", "https://api.prod.example"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.apiKey != "sk-token" || got.withToken {
		t.Fatalf("unexpected login args: %#v", got)
	}
	if got.targetContext != "prod" || got.targetServer != "https://api.prod.example" {
		t.Fatalf("unexpected target selection: %#v", got)
	}
}

func TestLoginCommand_DefaultsAndNonInteractiveError(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{serverURL: "https://api.example.com", timeout: time.Second}}
	forceAuthIsTerminal(t, func(int) bool { return false })

	cmd := newLoginCommand(state)
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "non-interactive mode") {
		t.Fatalf("expected non-interactive guidance, got: %v", err)
	}
}

func TestLoginCommand_DeviceFlowSelection(t *testing.T) {
	t.Parallel()

	t.Run("success returns immediately", func(t *testing.T) {
		state := &appState{opts: &rootOptions{serverURL: "https://api.example.com", timeout: time.Second}}
		forceAuthIsTerminal(t, func(int) bool { return true })

		deviceCalled := false
		apiKeyCalled := false
		forceAuthLoginWithDeviceCode(t, func(_ *cobra.Command, _ *appState, targetContext, targetServer string) error {
			deviceCalled = true
			if targetContext != "default" || targetServer != "https://api.example.com" {
				t.Fatalf("unexpected target selection: %s %s", targetContext, targetServer)
			}
			return nil
		})
		forceAuthLoginWithAPIKey(t, func(_ *cobra.Command, _ *appState, _ string, _ bool, _ string, _ string) error {
			apiKeyCalled = true
			return nil
		})

		cmd := newLoginCommand(state)
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !deviceCalled || apiKeyCalled {
			t.Fatalf("expected only device flow to run")
		}
	})

	t.Run("failure falls back to api key entry", func(t *testing.T) {
		state := &appState{opts: &rootOptions{serverURL: "https://api.example.com", timeout: time.Second}}
		forceAuthIsTerminal(t, func(int) bool { return true })

		apiKeyCalled := false
		forceAuthLoginWithDeviceCode(t, func(_ *cobra.Command, _ *appState, _, _ string) error {
			return errors.New("device unavailable")
		})
		forceAuthLoginWithAPIKey(t, func(_ *cobra.Command, _ *appState, apiKey string, withToken bool, targetContext, targetServer string) error {
			apiKeyCalled = true
			if apiKey != "" || withToken || targetContext != "default" || targetServer != "https://api.example.com" {
				t.Fatalf("unexpected fallback args: %q %t %q %q", apiKey, withToken, targetContext, targetServer)
			}
			return nil
		})

		stderr := captureCommandErrorOutput(t, func() {
			cmd := newLoginCommand(state)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		if !apiKeyCalled {
			t.Fatal("expected fallback api-key flow")
		}
		if !strings.Contains(stderr, "Device code login unavailable") || !strings.Contains(stderr, "Falling back to manual API key entry.") {
			t.Fatalf("expected fallback guidance, got: %s", stderr)
		}
	})

	t.Run("browser flag returns device error", func(t *testing.T) {
		state := &appState{opts: &rootOptions{serverURL: "https://api.example.com", timeout: time.Second}}
		forceAuthIsTerminal(t, func(int) bool { return true })

		forceAuthLoginWithDeviceCode(t, func(_ *cobra.Command, _ *appState, _, _ string) error {
			return errors.New("device unavailable")
		})
		forceAuthLoginWithAPIKey(t, func(_ *cobra.Command, _ *appState, _ string, _ bool, _ string, _ string) error {
			t.Fatal("manual api-key fallback should not run when --browser is explicit")
			return nil
		})

		cmd := newLoginCommand(state)
		cmd.SetArgs([]string{"--browser"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "device unavailable") {
			t.Fatalf("expected device flow error, got: %v", err)
		}
	})
}

func TestLoginWithAPIKey_SuccessOutputsAndConfig(t *testing.T) {
	t.Parallel()

	t.Run("json output", func(t *testing.T) {
		state := &appState{opts: &rootOptions{outputFormat: "json", timeout: time.Second}}
		state.configPath = testConfigPath(t)

		var savedContext, savedKey string
		forceAuthValidateAPIKey(t, func(_ context.Context, serverURL, apiKey string, timeout time.Duration) error {
			if serverURL != "https://api.example.com" || apiKey != "sk-json" || timeout != time.Second {
				t.Fatalf("unexpected validation inputs: %q %q %s", serverURL, apiKey, timeout)
			}
			return nil
		})
		forceAuthSaveAPIKey(t, func(contextName, apiKey string) error {
			savedContext, savedKey = contextName, apiKey
			return nil
		})

		out := captureCommandOutput(t, func() {
			if err := loginWithAPIKey(newAuthTestCommand(t), state, "sk-json", false, "prod", "https://api.example.com"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})

		if savedContext != "prod" || savedKey != "sk-json" {
			t.Fatalf("unexpected saved key: %q %q", savedContext, savedKey)
		}
		if !strings.Contains(out, `"authenticated": true`) || !strings.Contains(out, `"context": "prod"`) {
			t.Fatalf("expected JSON auth output, got: %s", out)
		}

		cfg := loadSavedConfig(t, state.configPath)
		if cfg.ActiveContext != "prod" || cfg.Contexts["prod"].Server != "https://api.example.com" {
			t.Fatalf("unexpected saved config: %#v", cfg)
		}
	})

	t.Run("tty output", func(t *testing.T) {
		state := &appState{opts: &rootOptions{timeout: time.Second}}
		state.configPath = testConfigPath(t)
		state.opts.outputFormat = ""
		forceStdoutTTY(t, true)
		forceAuthValidateAPIKey(t, func(_ context.Context, _, _ string, _ time.Duration) error { return nil })
		forceAuthSaveAPIKey(t, func(_, _ string) error { return nil })

		stderr := captureCommandErrorOutput(t, func() {
			if err := loginWithAPIKey(newAuthTestCommand(t), state, "sk-tty", false, "default", "https://api.example.com"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		if !strings.Contains(stderr, "Authenticated successfully") || !strings.Contains(stderr, "Context") {
			t.Fatalf("expected tty auth output, got: %s", stderr)
		}
	})
}

func TestLoginWithAPIKey_Errors(t *testing.T) {
	t.Parallel()

	t.Run("validation error", func(t *testing.T) {
		state := &appState{opts: &rootOptions{timeout: time.Second}}
		forceAuthValidateAPIKey(t, func(_ context.Context, _, _ string, _ time.Duration) error {
			return errors.New("invalid token")
		})

		err := loginWithAPIKey(newAuthTestCommand(t), state, "sk-bad", false, "default", "https://api.example.com")
		if err == nil || !strings.Contains(err.Error(), "invalid token") {
			t.Fatalf("expected validation error, got: %v", err)
		}
	})

	t.Run("save key error", func(t *testing.T) {
		state := &appState{opts: &rootOptions{timeout: time.Second}}
		forceAuthValidateAPIKey(t, func(_ context.Context, _, _ string, _ time.Duration) error { return nil })
		forceAuthSaveAPIKey(t, func(_, _ string) error {
			return errors.New("keyring unavailable")
		})

		err := loginWithAPIKey(newAuthTestCommand(t), state, "sk-bad", false, "default", "https://api.example.com")
		if err == nil || !strings.Contains(err.Error(), "save api key") {
			t.Fatalf("expected save error, got: %v", err)
		}
	})
}

func TestLoginWithDeviceCode_Behaviors(t *testing.T) {
	t.Parallel()

	t.Run("invalid server URL", func(t *testing.T) {
		state := &appState{opts: &rootOptions{timeout: time.Second}}
		err := loginWithDeviceCode(newAuthTestCommand(t), state, "default", ":")
		if err == nil || !strings.Contains(err.Error(), "create client") {
			t.Fatalf("expected client creation error, got: %v", err)
		}
	})

	t.Run("invalid verification URL", func(t *testing.T) {
		srv := newRouterServer(t, map[string]http.HandlerFunc{
			"POST /v1/cli/auth/device-code": func(w http.ResponseWriter, _ *http.Request) {
				respondJSON(t, w, http.StatusOK, map[string]any{
					"device_code":      "dev-1",
					"user_code":        "ABCD-EFGH",
					"verification_url": "://bad",
					"interval":         1,
					"expires_in":       30,
				})
			},
		})

		state := newTestState(t, srv)
		err := loginWithDeviceCode(newAuthTestCommand(t), state, "default", srv.URL)
		if err == nil || !strings.Contains(err.Error(), "invalid verification URL") {
			t.Fatalf("expected invalid verification URL error, got: %v", err)
		}
	})

	t.Run("open browser failure still succeeds with json output", func(t *testing.T) {
		var tokenRequests int
		srv := newRouterServer(t, map[string]http.HandlerFunc{
			"POST /v1/cli/auth/device-code": func(w http.ResponseWriter, _ *http.Request) {
				respondJSON(t, w, http.StatusOK, map[string]any{
					"device_code":      "dev-2",
					"user_code":        "ZXCV-BNMQ",
					"verification_url": "https://auth.example.net/verify",
					"interval":         1,
					"expires_in":       30,
				})
			},
			"POST /v1/cli/auth/token": func(w http.ResponseWriter, _ *http.Request) {
				tokenRequests++
				respondJSON(t, w, http.StatusOK, map[string]any{
					"api_key":    "sk-device",
					"project_id": "proj-from-device",
				})
			},
		})

		state := newTestState(t, srv)
		state.configPath = testConfigPath(t)
		var savedContext, savedKey string
		forceAuthSaveAPIKey(t, func(contextName, apiKey string) error {
			savedContext, savedKey = contextName, apiKey
			return nil
		})
		prevOpen := openBrowserFunc
		openBrowserFunc = func(string) error { return errors.New("no browser") }
		t.Cleanup(func() { openBrowserFunc = prevOpen })

		stdout, stderr := captureCommandStreams(t, func() {
			if err := loginWithDeviceCode(newAuthTestCommand(t), state, "device", srv.URL); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})

		if tokenRequests != 1 || savedContext != "device" || savedKey != "sk-device" {
			t.Fatalf("unexpected device save state: requests=%d context=%q key=%q", tokenRequests, savedContext, savedKey)
		}
		if !strings.Contains(stdout, `"authenticated": true`) || !strings.Contains(stdout, `"context": "device"`) {
			t.Fatalf("expected JSON device auth output, got stdout=%s stderr=%s", stdout, stderr)
		}
		if !strings.Contains(stderr, "warning: verification URL host") || !strings.Contains(stderr, "Could not open browser automatically") {
			t.Fatalf("expected mismatch and browser fallback messages, got: %s", stderr)
		}

		cfg := loadSavedConfig(t, state.configPath)
		if cfg.ActiveContext != "device" || cfg.Contexts["device"].Project != "proj-from-device" {
			t.Fatalf("unexpected saved config: %#v", cfg)
		}
	})

	t.Run("open browser success with tty output", func(t *testing.T) {
		var serverURL string
		srv := newRouterServer(t, map[string]http.HandlerFunc{
			"POST /v1/cli/auth/device-code": func(w http.ResponseWriter, _ *http.Request) {
				respondJSON(t, w, http.StatusOK, map[string]any{
					"device_code":      "dev-3",
					"user_code":        "QWER-TYUI",
					"verification_url": serverURL + "/verify",
					"interval":         1,
					"expires_in":       30,
				})
			},
			"POST /v1/cli/auth/token": func(w http.ResponseWriter, _ *http.Request) {
				respondJSON(t, w, http.StatusOK, map[string]any{
					"api_key":    "sk-tty-device",
					"project_id": "proj-tty-device",
				})
			},
		})
		serverURL = srv.URL

		state := newTestState(t, srv)
		state.configPath = testConfigPath(t)
		state.opts.outputFormat = ""
		forceStdoutTTY(t, true)
		forceAuthSaveAPIKey(t, func(_, _ string) error { return nil })
		prevOpen := openBrowserFunc
		openBrowserFunc = func(string) error { return nil }
		t.Cleanup(func() { openBrowserFunc = prevOpen })

		stderr := captureCommandErrorOutput(t, func() {
			if err := loginWithDeviceCode(newAuthTestCommand(t), state, "tty", srv.URL); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})

		if !strings.Contains(stderr, "Waiting for browser authorization...") || !strings.Contains(stderr, "Authenticated via device code") {
			t.Fatalf("expected tty device auth output, got: %s", stderr)
		}
	})

	t.Run("expired device code prints guidance", func(t *testing.T) {
		var serverURL string
		srv := newRouterServer(t, map[string]http.HandlerFunc{
			"POST /v1/cli/auth/device-code": func(w http.ResponseWriter, _ *http.Request) {
				respondJSON(t, w, http.StatusOK, map[string]any{
					"device_code":      "dev-4",
					"user_code":        "EXPR-IRED",
					"verification_url": serverURL + "/verify",
					"interval":         1,
					"expires_in":       30,
				})
			},
			"POST /v1/cli/auth/token": func(w http.ResponseWriter, _ *http.Request) {
				respondJSON(t, w, http.StatusBadRequest, map[string]any{"error": "expired_token"})
			},
		})
		serverURL = srv.URL

		state := newTestState(t, srv)
		prevOpen := openBrowserFunc
		openBrowserFunc = func(string) error { return nil }
		t.Cleanup(func() { openBrowserFunc = prevOpen })

		stderr := captureCommandErrorOutput(t, func() {
			err := loginWithDeviceCode(newAuthTestCommand(t), state, "default", srv.URL)
			if err == nil || !strings.Contains(err.Error(), "device code authorization") {
				t.Fatalf("expected device code auth error, got: %v", err)
			}
		})
		if !strings.Contains(stderr, "Authorization timed out.") {
			t.Fatalf("expected timeout guidance, got: %s", stderr)
		}
	})

	t.Run("interval zero path respects canceled context", func(t *testing.T) {
		var serverURL string
		srv := newRouterServer(t, map[string]http.HandlerFunc{
			"POST /v1/cli/auth/device-code": func(w http.ResponseWriter, _ *http.Request) {
				respondJSON(t, w, http.StatusOK, map[string]any{
					"device_code":      "dev-5",
					"user_code":        "CANC-ELLD",
					"verification_url": serverURL + "/verify",
					"interval":         0,
					"expires_in":       30,
				})
			},
		})
		serverURL = srv.URL

		state := newTestState(t, srv)
		cmd := newAuthTestCommand(t)
		ctx, cancel := context.WithCancel(t.Context())
		cmd.SetContext(ctx)
		prevOpen := openBrowserFunc
		openBrowserFunc = func(string) error {
			cancel()
			return nil
		}
		t.Cleanup(func() { openBrowserFunc = prevOpen })

		err := loginWithDeviceCode(cmd, state, "default", srv.URL)
		if err == nil || !strings.Contains(err.Error(), "device code authorization") {
			t.Fatalf("expected canceled device auth error, got: %v", err)
		}
	})

	t.Run("save key error", func(t *testing.T) {
		var serverURL string
		srv := newRouterServer(t, map[string]http.HandlerFunc{
			"POST /v1/cli/auth/device-code": func(w http.ResponseWriter, _ *http.Request) {
				respondJSON(t, w, http.StatusOK, map[string]any{
					"device_code":      "dev-6",
					"user_code":        "SAVE-FAIL",
					"verification_url": serverURL + "/verify",
					"interval":         1,
					"expires_in":       30,
				})
			},
			"POST /v1/cli/auth/token": func(w http.ResponseWriter, _ *http.Request) {
				respondJSON(t, w, http.StatusOK, map[string]any{"api_key": "sk-device"})
			},
		})
		serverURL = srv.URL

		state := newTestState(t, srv)
		forceAuthSaveAPIKey(t, func(_, _ string) error {
			return errors.New("keyring unavailable")
		})
		prevOpen := openBrowserFunc
		openBrowserFunc = func(string) error { return nil }
		t.Cleanup(func() { openBrowserFunc = prevOpen })

		err := loginWithDeviceCode(newAuthTestCommand(t), state, "default", srv.URL)
		if err == nil || !strings.Contains(err.Error(), "save api key") {
			t.Fatalf("expected save error, got: %v", err)
		}
	})
}

func TestLogoutAndAuthStatus(t *testing.T) {
	t.Parallel()

	t.Run("logout json uses default context", func(t *testing.T) {
		state := &appState{opts: &rootOptions{outputFormat: "json", serverURL: "https://api.example.com"}}
		var deleted string
		forceAuthDeleteAPIKey(t, func(contextName string) error {
			deleted = contextName
			return nil
		})

		out := captureCommandOutput(t, func() {
			cmd := newLogoutCommand(state)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})

		if deleted != "default" || !strings.Contains(out, `"logged_out": true`) {
			t.Fatalf("unexpected logout output: deleted=%q out=%s", deleted, out)
		}
	})

	t.Run("logout tty and delete error", func(t *testing.T) {
		state := &appState{opts: &rootOptions{serverURL: "https://api.example.com", contextName: "staging"}}
		state.opts.outputFormat = ""
		forceStdoutTTY(t, true)

		forceAuthDeleteAPIKey(t, func(contextName string) error {
			if contextName != "staging" {
				t.Fatalf("unexpected context delete: %q", contextName)
			}
			return nil
		})
		stderr := captureCommandErrorOutput(t, func() {
			cmd := newLogoutCommand(state)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		if !strings.Contains(stderr, "Logged out from context") || !strings.Contains(stderr, "staging") {
			t.Fatalf("expected tty logout output, got: %s", stderr)
		}

		forceAuthDeleteAPIKey(t, func(string) error { return errors.New("keyring unavailable") })
		cmd := newLogoutCommand(state)
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "delete api key") {
			t.Fatalf("expected delete error, got: %v", err)
		}
	})

	t.Run("auth status json and tty", func(t *testing.T) {
		state := &appState{opts: &rootOptions{outputFormat: "json", serverURL: "https://api.example.com"}}
		forceAuthLoadAPIKey(t, func(string) (string, error) {
			return "", errors.New("not found")
		})

		jsonOut := captureCommandOutput(t, func() {
			cmd := newAuthCommand(state)
			cmd.SetArgs([]string{"status"})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		if !strings.Contains(jsonOut, `"authenticated": false`) || !strings.Contains(jsonOut, `"context": "default"`) {
			t.Fatalf("expected unauthenticated json, got: %s", jsonOut)
		}

		state.opts.contextName = "prod"
		state.opts.outputFormat = ""
		forceStdoutTTY(t, true)
		forceAuthLoadAPIKey(t, func(contextName string) (string, error) {
			if contextName != "prod" {
				t.Fatalf("unexpected load context: %q", contextName)
			}
			return "sk-prod", nil
		})
		stderr := captureCommandErrorOutput(t, func() {
			cmd := newAuthCommand(state)
			cmd.SetArgs([]string{"status"})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		if !strings.Contains(stderr, "Authenticated") || !strings.Contains(stderr, "Context") || !strings.Contains(stderr, "Server") {
			t.Fatalf("expected tty auth status output, got: %s", stderr)
		}
	})
}

func TestResolveAPIKeyInput_Behaviors(t *testing.T) {
	t.Run("explicit flag wins", func(t *testing.T) {
		got, err := resolveAPIKeyInput("  sk-flag  ", false)
		if err != nil || got != "sk-flag" {
			t.Fatalf("expected trimmed explicit flag, got %q err=%v", got, err)
		}
	})

	t.Run("stdin token", func(t *testing.T) {
		withMockStdin(t, "sk-stdin\n", func() {
			got, err := resolveAPIKeyInput("", true)
			if err != nil || got != "sk-stdin" {
				t.Fatalf("expected stdin token, got %q err=%v", got, err)
			}
		})
	})

	t.Run("stdin read error", func(t *testing.T) {
		withClosedStdin(t, func() {
			_, err := resolveAPIKeyInput("", true)
			if err == nil {
				t.Fatal("expected stdin read error")
			}
		})
	})

	t.Run("env fallback", func(t *testing.T) {
		t.Setenv("STRAIT_API_KEY", "sk-env")
		got, err := resolveAPIKeyInput("", false)
		if err != nil || got != "sk-env" {
			t.Fatalf("expected env token, got %q err=%v", got, err)
		}
	})

	t.Run("prompt success", func(t *testing.T) {
		forceAuthReadSecret(t, func(fd int) ([]byte, error) {
			if fd != syscall.Stdin {
				t.Fatalf("unexpected fd: %d", fd)
			}
			return []byte("  sk-prompt  "), nil
		})
		stderr := captureCommandErrorOutput(t, func() {
			got, err := resolveAPIKeyInput("", false)
			if err != nil || got != "sk-prompt" {
				t.Fatalf("expected prompt token, got %q err=%v", got, err)
			}
		})
		if !strings.Contains(stderr, "API key:") {
			t.Fatalf("expected prompt output, got: %s", stderr)
		}
	})

	t.Run("prompt error", func(t *testing.T) {
		forceAuthReadSecret(t, func(int) ([]byte, error) {
			return nil, errors.New("read failed")
		})
		_, err := resolveAPIKeyInput("", false)
		if err == nil || !strings.Contains(err.Error(), "read failed") {
			t.Fatalf("expected prompt read error, got: %v", err)
		}
	})

	t.Run("prompt empty requires key", func(t *testing.T) {
		forceAuthReadSecret(t, func(int) ([]byte, error) {
			return []byte("   "), nil
		})
		_, err := resolveAPIKeyInput("", false)
		if err == nil || !strings.Contains(err.Error(), "api key is required") {
			t.Fatalf("expected missing key error, got: %v", err)
		}
	})
}
