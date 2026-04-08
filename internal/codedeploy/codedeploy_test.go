package codedeploy

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestFormatSize(t *testing.T) {
	t.Parallel()
	cases := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{2048, "2.0 KB"},
		{1024 * 1024, "1.0 MB"},
		{5 * 1024 * 1024, "5.0 MB"},
	}
	for _, tc := range cases {
		got := formatSize(tc.bytes)
		if got != tc.want {
			t.Errorf("formatSize(%d) = %q, want %q", tc.bytes, got, tc.want)
		}
	}
}

func TestProgressReader_CallsCallbackWithCumulativeBytes(t *testing.T) {
	t.Parallel()

	data := []byte("hello world this is test data for the progress reader")
	total := int64(len(data))
	r := &progressReader{
		r:     bytes.NewReader(data),
		total: total,
		onProgress: func(read, tot int64) {
			if tot != total {
				t.Errorf("progress total: got %d, want %d", tot, total)
			}
			if read > total {
				t.Errorf("read (%d) exceeds total (%d)", read, total)
			}
		},
	}

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != string(data) {
		t.Errorf("data mismatch: got %q, want %q", out, data)
	}
	if r.read != total {
		t.Errorf("final read count: got %d, want %d", r.read, total)
	}
}

func TestProgressReader_SmallReadsAccumulate(t *testing.T) {
	t.Parallel()

	data := make([]byte, 100)
	for i := range data {
		data[i] = byte(i)
	}

	var lastRead int64
	r := &progressReader{
		r:     bytes.NewReader(data),
		total: int64(len(data)),
		onProgress: func(read, _ int64) {
			if read < lastRead {
				t.Errorf("read decreased: %d < %d", read, lastRead)
			}
			lastRead = read
		},
	}

	// Read in 10-byte chunks to exercise small-read accumulation.
	buf := make([]byte, 10)
	for {
		n, err := r.Read(buf)
		if n == 0 && errors.Is(err, io.EOF) {
			break
		}
		if err != nil && !errors.Is(err, io.EOF) {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if r.read != int64(len(data)) {
		t.Errorf("final read: got %d, want %d", r.read, len(data))
	}
}
