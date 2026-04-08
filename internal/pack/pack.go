// Package pack creates a gzipped tar archive of a source directory.
// It respects .straitignore (gitignore-style) pattern files and always
// skips .git, .straitignore itself, and the output archive if it lives
// inside the source directory.
package pack

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Result holds the output of a Pack call.
type Result struct {
	// Path is the absolute path to the temp file containing the tar.gz archive.
	// The caller is responsible for removing it when done.
	Path string
	// Hash is the hex-encoded SHA-256 digest of the archive content.
	Hash string
	// Size is the byte size of the archive.
	Size int64
}

// Pack walks dir, applies ignore rules, and writes a deterministic gzipped tar
// to a temporary file. The caller is responsible for deleting the temp file.
func Pack(dir string, ignoreFile string) (*Result, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolve source dir: %w", err)
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("stat source dir %s: %w", abs, err)
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", abs)
	}

	patterns := defaultIgnorePatterns()
	if ignoreFile != "" {
		extra, readErr := loadIgnoreFile(ignoreFile)
		if readErr != nil {
			return nil, fmt.Errorf("read ignore file: %w", readErr)
		}
		patterns = append(patterns, extra...)
	} else {
		// Auto-discover .straitignore in source dir.
		candidate := filepath.Join(abs, ".straitignore")
		if _, statErr := os.Stat(candidate); statErr == nil {
			extra, readErr := loadIgnoreFile(candidate)
			if readErr != nil {
				return nil, fmt.Errorf("read .straitignore: %w", readErr)
			}
			patterns = append(patterns, extra...)
		}
	}

	tmp, err := os.CreateTemp("", "strait-pack-*.tar.gz")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}

	h := sha256.New()
	mw := io.MultiWriter(tmp, h)

	if writeErr := writeArchive(mw, abs, patterns); writeErr != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return nil, writeErr
	}

	size, err := tmp.Seek(0, io.SeekCurrent)
	if err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return nil, fmt.Errorf("get archive size: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return nil, fmt.Errorf("close temp file: %w", err)
	}

	return &Result{
		Path: tmp.Name(),
		Hash: hex.EncodeToString(h.Sum(nil)),
		Size: size,
	}, nil
}

// writeArchive creates a deterministic gzipped tar from dir into w.
func writeArchive(w io.Writer, dir string, patterns []ignorePattern) error {
	gz := gzip.NewWriter(w)
	tw := tar.NewWriter(gz)

	if err := filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		if rel == "." {
			return nil
		}

		if shouldIgnore(rel, d.IsDir(), patterns) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		// Only include regular files and directories; skip symlinks, devices, etc.
		mode := info.Mode()
		if !mode.IsDir() && !mode.IsRegular() {
			return nil
		}

		hdr := &tar.Header{
			Name:     rel,
			Mode:     int64(mode.Perm()),
			ModTime:  info.ModTime().UTC().Truncate(1),
			Typeflag: tar.TypeReg,
		}
		if mode.IsDir() {
			hdr.Typeflag = tar.TypeDir
			hdr.Name = rel + "/"
			return tw.WriteHeader(hdr)
		}

		hdr.Size = info.Size()
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		f, err := os.Open(path) //nolint:gosec // path is always rooted at dir; symlinks are skipped above
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err := io.Copy(tw, f); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return fmt.Errorf("walk source dir: %w", err)
	}

	if err := tw.Close(); err != nil {
		return fmt.Errorf("close tar: %w", err)
	}
	return gz.Close()
}

// ignorePattern holds a parsed ignore rule.
type ignorePattern struct {
	raw      string // original pattern (for debugging)
	segments []string
	anchored bool // true if pattern contains a slash before the last segment
	dirOnly  bool // true if pattern ends with /
	negate   bool // true if pattern starts with !
}

// defaultIgnorePatterns returns patterns that are always excluded.
func defaultIgnorePatterns() []ignorePattern {
	defaults := []string{
		".git",
		".git/",
		".straitignore",
		".DS_Store",
		"Thumbs.db",
	}
	out := make([]ignorePattern, 0, len(defaults))
	for _, p := range defaults {
		out = append(out, parsePattern(p))
	}
	return out
}

// loadIgnoreFile reads an ignore file and returns parsed patterns.
func loadIgnoreFile(path string) ([]ignorePattern, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path is user-supplied
	if err != nil {
		return nil, err
	}
	var patterns []ignorePattern
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, parsePattern(line))
	}
	return patterns, nil
}

// parsePattern converts a raw gitignore-style pattern to an ignorePattern.
func parsePattern(raw string) ignorePattern {
	p := ignorePattern{raw: raw}
	s := raw

	if strings.HasPrefix(s, "!") {
		p.negate = true
		s = s[1:]
	}

	// Strip leading /
	if strings.HasPrefix(s, "/") {
		s = s[1:]
		p.anchored = true
	}

	// Trailing / means directory-only match.
	if strings.HasSuffix(s, "/") {
		p.dirOnly = true
		s = strings.TrimSuffix(s, "/")
	}

	// If pattern contains a /, it's anchored.
	if strings.Contains(s, "/") {
		p.anchored = true
	}

	p.segments = strings.Split(s, "/")
	return p
}

// shouldIgnore returns true if the path should be excluded.
// rel is the path relative to the archive root (slash-separated).
// isDir indicates whether the path is a directory.
func shouldIgnore(rel string, isDir bool, patterns []ignorePattern) bool {
	matched := false
	for _, p := range patterns {
		if p.dirOnly && !isDir {
			continue
		}
		m := matchPattern(rel, isDir, p)
		if m {
			if p.negate {
				matched = false
			} else {
				matched = true
			}
		}
	}
	return matched
}

// matchPattern checks whether rel matches pattern p.
func matchPattern(rel string, _ bool, p ignorePattern) bool {
	relParts := strings.Split(rel, "/")

	if !p.anchored {
		// Unanchored: can match any suffix of the path.
		for i := range relParts {
			if matchSegments(relParts[i:], p.segments) {
				return true
			}
		}
		// Also match top-level without suffix sliding for single-segment patterns.
		return matchSegments(relParts, p.segments)
	}

	return matchSegments(relParts, p.segments)
}

// matchSegments matches path segments against pattern segments, supporting *.
// ** in a pattern segment matches zero or more path segments.
func matchSegments(pathParts, patParts []string) bool {
	if len(patParts) == 0 && len(pathParts) == 0 {
		return true
	}
	if len(patParts) == 0 {
		return false
	}

	if patParts[0] == "**" {
		// ** matches zero or more segments.
		// Try matching the rest of the pattern against each suffix.
		for i := 0; i <= len(pathParts); i++ {
			if matchSegments(pathParts[i:], patParts[1:]) {
				return true
			}
		}
		return false
	}

	if len(pathParts) == 0 {
		return false
	}

	matched, err := filepath.Match(patParts[0], pathParts[0])
	if err != nil || !matched {
		return false
	}

	if len(patParts) == 1 {
		// Pattern consumed: match if this is the last path part or pattern
		// is a prefix match (e.g., "foo" should match "foo" and "foo/bar").
		return true
	}

	return matchSegments(pathParts[1:], patParts[1:])
}
