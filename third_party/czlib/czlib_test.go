package czlib

import (
	"bytes"
	"compress/zlib"
	"io"
	"testing"
)

func TestNewReaderRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write([]byte("strata")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	r, err := NewReader(&buf)
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "strata" {
		t.Fatalf("got %q", got)
	}
}
