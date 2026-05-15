package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	cliauth "github.com/strait-dev/cli/internal/auth"
	"github.com/strait-dev/cli/internal/client"
	cliconfig "github.com/strait-dev/cli/internal/config"
	"github.com/strait-dev/cli/internal/styles"

	"github.com/spf13/cobra"
)

var (
	authIsTerminal          = stdinIsTerminal
	authReadSecret          = readStdinPassword
	authValidateAPIKey      = cliauth.ValidateAPIKey
	authSaveAPIKey          = cliauth.SaveAPIKey
	authLoadAPIKey          = cliauth.LoadAPIKey
	authDeleteAPIKey        = cliauth.DeleteAPIKey
	authLoginWithDeviceCode = loginWithDeviceCode
	authLoginWithAPIKey     = loginWithAPIKey
)

func newLoginCommand(state *appState) *cobra.Command {
	var withToken bool
	var token string
	var contextName string
	var server string
	var browser bool
	var noBrowser bool

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with the Strait API",
		Long: `Authenticate with the Strait API using browser-based device code flow or a direct API key.

By default, opens a browser for device code authorization. Use --token to provide
an API key directly, or --with-token to read one from stdin.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			targetContext := contextName
			if targetContext == "" {
				targetContext = state.opts.contextName
			}
			if targetContext == "" {
				targetContext = "default"
			}

			targetServer := server
			if targetServer == "" {
				targetServer = state.opts.serverURL
			}

			// Direct token mode: --token or --with-token provided.
			if token != "" || withToken {
				return authLoginWithAPIKey(cmd, state, token, withToken, targetContext, targetServer)
			}

			// Non-interactive or non-TTY without explicit token: error with guidance.
			if state.opts.nonInteractive || !authIsTerminal() {
				return fmt.Errorf("non-interactive mode: use --token <api-key> or STRAIT_API_KEY env var to authenticate")
			}

			// Browser-based device code flow (default for interactive terminals).
			useDeviceFlow := !noBrowser
			if useDeviceFlow {
				err := authLoginWithDeviceCode(cmd, state, targetContext, targetServer)
				if err == nil {
					return nil
				}
				// If device code flow fails, fall back to manual key entry
				// unless --browser was explicitly set (user expected it to work).
				if browser {
					return err
				}
				fmt.Fprintf(os.Stderr, "Device code login unavailable: %v\n", err)
				fmt.Fprintln(os.Stderr, "Falling back to manual API key entry.")
			}

			return authLoginWithAPIKey(cmd, state, "", false, targetContext, targetServer)
		},
	}

	cmd.Flags().BoolVar(&withToken, "with-token", false, "read API key from stdin")
	cmd.Flags().StringVar(&token, "token", "", "API key for direct authentication")
	cmd.Flags().StringVar(&contextName, "context", "", "context to save API key under")
	cmd.Flags().StringVar(&server, "server", "", "server URL to validate against")
	cmd.Flags().BoolVar(&browser, "browser", false, "open the dashboard API key page in your browser")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "do not open browser")

	return cmd
}

// loginWithDeviceCode performs the OAuth device code authorization flow.
func loginWithDeviceCode(cmd *cobra.Command, state *appState, targetContext, targetServer string) error {
	c, err := client.New(targetServer, "", state.opts.timeout)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}

	resp, err := c.RequestDeviceCode(cmd.Context())
	if err != nil {
		return fmt.Errorf("request device code: %w", err)
	}

	// Validate the verification URL before showing or opening it.
	verifURL, parseErr := url.Parse(resp.VerificationURL)
	if parseErr != nil || (verifURL.Scheme != "http" && verifURL.Scheme != "https") || verifURL.Host == "" {
		return fmt.Errorf("server returned invalid verification URL: %q", resp.VerificationURL)
	}
	// Verify the verification URL host matches the server we connected to.
	serverURL, _ := url.Parse(targetServer)
	if serverURL != nil && verifURL.Host != serverURL.Host &&
		verifURL.Host != strings.Replace(serverURL.Host, "api.", "app.", 1) &&
		verifURL.Host != strings.Replace(serverURL.Host, ":8080", ":5173", 1) {
		fmt.Fprintf(os.Stderr, "warning: verification URL host %q does not match server %q\n", verifURL.Host, serverURL.Host)
	}

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintf(os.Stderr, "Your one-time code: %s\n", resp.UserCode)
	fmt.Fprintf(os.Stderr, "Open this URL to authenticate: %s\n", resp.VerificationURL)
	fmt.Fprintln(os.Stderr, "")

	// Try to open the browser automatically.
	if err := openBrowserFunc(resp.VerificationURL); err != nil {
		fmt.Fprintf(os.Stderr, "Could not open browser automatically. Visit the URL above.\n")
	} else {
		fmt.Fprintln(os.Stderr, "Waiting for browser authorization...")
	}

	// Poll with progress indicator.
	ctx := cmd.Context()
	interval := resp.Interval
	if interval <= 0 {
		interval = 5
	}

	// Start a goroutine to print dots as a progress indicator.
	done := make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r != nil {
				_ = r // swallow panic in cosmetic progress goroutine
			}
		}()
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				fmt.Fprint(os.Stderr, ".")
			}
		}
	}()

	tokenResp, err := c.PollDeviceToken(ctx, resp.DeviceCode, interval, resp.ExpiresIn)
	close(done)
	fmt.Fprintln(os.Stderr, "") // newline after dots

	if err != nil {
		if strings.Contains(err.Error(), "expired") {
			fmt.Fprintln(os.Stderr, "Authorization timed out. You can try again or use --token <api-key> instead.")
		}
		return fmt.Errorf("device code authorization: %w", err)
	}

	if err := authSaveAPIKey(targetContext, tokenResp.APIKey); err != nil {
		return fmt.Errorf("save api key: %w", err)
	}

	cfg, cfgPath, err := loadConfigForWrite(state)
	if err != nil {
		return err
	}
	if cfg.Contexts == nil {
		cfg.Contexts = make(map[string]cliconfig.Context)
	}
	cfgCtx := cfg.Contexts[targetContext]
	if targetServer != "" {
		cfgCtx.Server = targetServer
	}
	if tokenResp.ProjectID != "" {
		cfgCtx.Project = tokenResp.ProjectID
	}
	cfg.Contexts[targetContext] = cfgCtx
	cfg.ActiveContext = targetContext
	if err := cliconfig.Save(cfgPath, cfg); err != nil {
		return err
	}

	if isTTYRich(state) {
		fmt.Fprintln(os.Stderr, styles.Success("Authenticated via device code"))
		fmt.Fprintln(os.Stderr, styles.KeyValue("Context", targetContext))
		return nil
	}
	return printData(state, map[string]any{
		"authenticated": true,
		"context":       targetContext,
		"server":        targetServer,
	})
}

// loginWithAPIKey handles direct API key authentication (--token, --api-key, --with-token, or manual entry).
func loginWithAPIKey(cmd *cobra.Command, state *appState, apiKey string, withToken bool, targetContext, targetServer string) error {
	resolvedKey, err := resolveAPIKeyInput(apiKey, withToken)
	if err != nil {
		return err
	}

	if err := authValidateAPIKey(cmd.Context(), targetServer, resolvedKey, state.opts.timeout); err != nil {
		return err
	}

	if err := authSaveAPIKey(targetContext, resolvedKey); err != nil {
		return fmt.Errorf("save api key: %w", err)
	}

	cfg, cfgPath, err := loadConfigForWrite(state)
	if err != nil {
		return err
	}
	if cfg.Contexts == nil {
		cfg.Contexts = make(map[string]cliconfig.Context)
	}
	cfgCtx := cfg.Contexts[targetContext]
	if targetServer != "" {
		cfgCtx.Server = targetServer
	}
	cfg.Contexts[targetContext] = cfgCtx
	cfg.ActiveContext = targetContext
	if err := cliconfig.Save(cfgPath, cfg); err != nil {
		return err
	}

	if isTTYRich(state) {
		fmt.Fprintln(os.Stderr, styles.Success("Authenticated successfully"))
		fmt.Fprintln(os.Stderr, styles.KeyValue("Context", targetContext))
		return nil
	}
	return printData(state, map[string]any{
		"authenticated": true,
		"context":       targetContext,
		"server":        targetServer,
	})
}

func newLogoutCommand(state *appState) *cobra.Command {
	var contextName string

	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Remove stored API key from keychain",
		RunE: func(_ *cobra.Command, _ []string) error {
			targetContext := contextName
			if targetContext == "" {
				targetContext = state.opts.contextName
			}
			if targetContext == "" {
				targetContext = "default"
			}

			if err := authDeleteAPIKey(targetContext); err != nil {
				return fmt.Errorf("delete api key: %w", err)
			}

			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Info("Logged out from context "+styles.Bold.Render(targetContext)))
				return nil
			}
			return printData(state, map[string]any{
				"logged_out": true,
				"context":    targetContext,
			})
		},
	}

	cmd.Flags().StringVar(&contextName, "context", "", "context to remove API key from")

	return cmd
}

func newAuthCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication helper commands",
	}

	cmd.AddCommand(newLoginCommand(state))
	cmd.AddCommand(newLogoutCommand(state))

	cmd.AddCommand(&cobra.Command{
		Use:   "whoami",
		Short: "Show authentication status",
		RunE: func(_ *cobra.Command, _ []string) error {
			targetContext := state.opts.contextName
			if targetContext == "" {
				targetContext = "default"
			}

			_, err := authLoadAPIKey(targetContext)
			authed := err == nil

			if isTTYRich(state) {
				if authed {
					fmt.Fprintln(os.Stderr, styles.Success("Authenticated"))
				} else {
					fmt.Fprintln(os.Stderr, styles.Warn("Not authenticated"))
				}
				fmt.Fprintln(os.Stderr, styles.KeyValue("Context", targetContext))
				fmt.Fprintln(os.Stderr, styles.KeyValue("Server", state.opts.serverURL))
				return nil
			}
			return printData(state, map[string]any{
				"authenticated": authed,
				"context":       targetContext,
				"server":        state.opts.serverURL,
			})
		},
	})

	return cmd
}

func resolveAPIKeyInput(flagValue string, withToken bool) (string, error) {
	if v := strings.TrimSpace(flagValue); v != "" {
		return v, nil
	}

	if withToken {
		reader := bufio.NewReader(os.Stdin)
		token, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", err
		}
		if v := strings.TrimSpace(token); v != "" {
			return v, nil
		}
	}

	if v := strings.TrimSpace(os.Getenv("STRAIT_API_KEY")); v != "" {
		return v, nil
	}

	fmt.Fprint(os.Stderr, "API key: ")
	secret, err := authReadSecret()
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	if v := strings.TrimSpace(string(secret)); v != "" {
		return v, nil
	}

	return "", fmt.Errorf("api key is required")
}
