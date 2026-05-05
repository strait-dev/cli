package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const updateCheckCacheDuration = 24 * time.Hour

// githubReleasesURL is a var (not const) so tests can override it.
var githubReleasesURL = "https://api.github.com/repos/strait-dev/cli/releases/latest"

type updateCheckCache struct {
	LatestVersion string    `json:"latest_version"`
	CheckedAt     time.Time `json:"checked_at"`
}

// checkForUpdate queries GitHub releases API for the latest version.
// Returns the latest version tag or empty string on error.
func checkForUpdate() string {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(githubReleasesURL) //nolint:noctx // fire-and-forget background check
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	// Limit read to 1 MB — the GitHub releases API should never return more.
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1024*1024)).Decode(&release); err != nil {
		return ""
	}

	return strings.TrimPrefix(release.TagName, "v")
}

// getCachedUpdate returns the cached latest version if the cache is fresh.
func getCachedUpdate() (string, bool) {
	cachePath := updateCachePath()
	if cachePath == "" {
		return "", false
	}

	data, err := os.ReadFile(cachePath) //nolint:gosec // cache file from known path
	if err != nil {
		return "", false
	}

	var cache updateCheckCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return "", false
	}

	if time.Since(cache.CheckedAt) > updateCheckCacheDuration {
		return "", false
	}

	return cache.LatestVersion, true
}

// setCachedUpdate writes the latest version to the cache file.
func setCachedUpdate(latestVersion string) {
	cachePath := updateCachePath()
	if cachePath == "" {
		return
	}

	cache := updateCheckCache{
		LatestVersion: latestVersion,
		CheckedAt:     time.Now(),
	}
	data, err := json.Marshal(cache)
	if err != nil {
		return
	}

	dir := filepath.Dir(cachePath)
	_ = os.MkdirAll(dir, 0o750)
	_ = os.WriteFile(cachePath, data, 0o644) //nolint:gosec // cache file with standard permissions
}

func updateCachePath() string {
	if dir := os.Getenv("XDG_CACHE_HOME"); dir != "" {
		return filepath.Join(dir, "strait", "update-check.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".cache", "strait", "update-check.json")
}

func newUpgradeCommand(state *appState) *cobra.Command {
	var apply bool

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Check for CLI updates and optionally self-update",
		Long: `Checks GitHub releases for a newer version of the Strait CLI.
With --apply, downloads and replaces the current binary in place.`,
		Example: `  strait upgrade
  strait upgrade --apply`,
		RunE: func(_ *cobra.Command, _ []string) error {
			latest := checkForUpdate()
			if latest == "" {
				return fmt.Errorf("failed to check for updates")
			}

			setCachedUpdate(latest)

			w := state.out()
			current := strings.TrimPrefix(version, "v")
			if current == latest {
				fmt.Fprintf(w, "Already up to date (v%s)\n", current)
				return nil
			}

			fmt.Fprintf(w, "Current: v%s\nLatest:  v%s\n", current, latest)

			if !apply {
				fmt.Fprintln(w, "\nTo upgrade, run: strait upgrade --apply")
				fmt.Fprintf(w, "Or download from: https://github.com/strait-dev/cli/releases/tag/v%s\n", latest)
				return nil
			}

			return selfUpdate(latest)
		},
	}

	cmd.Flags().BoolVar(&apply, "apply", false, "download and replace the current binary")

	return cmd
}

// selfUpdate downloads the latest release and replaces the current binary.
func selfUpdate(version string) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("detect binary path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("resolve binary path: %w", err)
	}

	goos := runtime.GOOS
	goarch := runtime.GOARCH
	archiveName := fmt.Sprintf("strait_%s_%s_%s.tar.gz", version, goos, goarch)
	if goos == "windows" {
		archiveName = fmt.Sprintf("strait_%s_%s_%s.zip", version, goos, goarch)
	}

	downloadURL := fmt.Sprintf("https://github.com/strait-dev/cli/releases/download/v%s/%s", version, archiveName)
	fmt.Fprintf(os.Stderr, "Downloading %s...\n", downloadURL)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Get(downloadURL) //nolint:noctx // short-lived CLI command
	if err != nil {
		return fmt.Errorf("download release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	// Read the archive into a temp file in the same directory as the binary
	// so os.Rename works (same filesystem).
	dir := filepath.Dir(execPath)
	tmpFile, err := os.CreateTemp(dir, "strait-upgrade-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w (try running with elevated permissions)", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Extract the "strait" binary from the tar.gz archive.
	binary, err := extractBinaryFromTarGz(resp.Body, "strait")
	if err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("extract binary: %w", err)
	}

	if _, err := tmpFile.Write(binary); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("write binary: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	// Make executable.
	if err := os.Chmod(tmpPath, 0o755); err != nil { //nolint:gosec // binary must be executable
		return fmt.Errorf("chmod: %w", err)
	}

	// Atomic rename to replace the current binary.
	if err := os.Rename(tmpPath, execPath); err != nil {
		return fmt.Errorf("replace binary: %w (try running with elevated permissions)", err)
	}

	fmt.Fprintf(os.Stderr, "Upgraded to v%s\n", version)
	return nil
}

// extractBinaryFromTarGz reads a tar.gz archive and returns the contents of
// the file matching binaryName.
func extractBinaryFromTarGz(r io.Reader, binaryName string) ([]byte, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("open gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read tar entry: %w", err)
		}

		name := filepath.Base(hdr.Name)
		if name == binaryName && hdr.Typeflag == tar.TypeReg {
			// Limit read to 200MB to prevent resource exhaustion.
			const maxBinarySize = 200 * 1024 * 1024
			data, err := io.ReadAll(io.LimitReader(tr, maxBinarySize))
			if err != nil {
				return nil, fmt.Errorf("read binary: %w", err)
			}
			return data, nil
		}
	}

	return nil, fmt.Errorf("binary %q not found in archive", binaryName)
}
