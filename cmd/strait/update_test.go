package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"testing"
)

func TestReleaseArchiveName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		goos   string
		goarch string
		want   string
	}{
		{name: "linux tarball", goos: "linux", goarch: "amd64", want: "strait_1.2.3_linux_amd64.tar.gz"},
		{name: "darwin tarball", goos: "darwin", goarch: "arm64", want: "strait_1.2.3_darwin_arm64.tar.gz"},
		{name: "windows zip", goos: "windows", goarch: "amd64", want: "strait_1.2.3_windows_amd64.zip"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := releaseArchiveName("1.2.3", tc.goos, tc.goarch)
			if got != tc.want {
				t.Fatalf("releaseArchiveName() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestChecksumForAsset(t *testing.T) {
	t.Parallel()

	asset := []byte("binary")
	sum := sha256.Sum256(asset)
	checksums := fmt.Appendf(nil, "%x  strait_1.2.3_linux_amd64.tar.gz\n", sum)

	got, err := checksumForAsset(checksums, "strait_1.2.3_linux_amd64.tar.gz")
	if err != nil {
		t.Fatalf("checksumForAsset: %v", err)
	}
	if got != fmt.Sprintf("%x", sum) {
		t.Fatalf("checksum = %q, want %x", got, sum)
	}
}

func TestChecksumForAsset_Missing(t *testing.T) {
	t.Parallel()

	_, err := checksumForAsset([]byte("abc  other.tar.gz\n"), "strait.tar.gz")
	if err == nil {
		t.Fatal("expected missing checksum error")
	}
}

func TestExtractBinaryFromArchive_TarGz(t *testing.T) {
	t.Parallel()

	archive := buildTarGz(t, "dist/strait", []byte("tar-binary"))
	got, err := extractBinaryFromArchive(archive, "strait_1.2.3_linux_amd64.tar.gz", "strait")
	if err != nil {
		t.Fatalf("extractBinaryFromArchive: %v", err)
	}
	if string(got) != "tar-binary" {
		t.Fatalf("binary = %q", got)
	}
}

func TestExtractBinaryFromArchive_Zip(t *testing.T) {
	t.Parallel()

	archive := buildZip(t, "dist/strait.exe", []byte("zip-binary"))
	got, err := extractBinaryFromArchive(archive, "strait_1.2.3_windows_amd64.zip", "strait.exe")
	if err != nil {
		t.Fatalf("extractBinaryFromArchive: %v", err)
	}
	if string(got) != "zip-binary" {
		t.Fatalf("binary = %q", got)
	}
}

func buildTarGz(t *testing.T, name string, data []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{
		Name: name,
		Mode: 0o755,
		Size: int64(len(data)),
	}); err != nil {
		t.Fatalf("write tar header: %v", err)
	}
	if _, err := tw.Write(data); err != nil {
		t.Fatalf("write tar data: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	return buf.Bytes()
}

func buildZip(t *testing.T, name string, data []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(name)
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	if _, err := w.Write(data); err != nil {
		t.Fatalf("write zip data: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buf.Bytes()
}
