package commands

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareUploadSourceLocalFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cat.txt")
	content := []byte("hello from disk")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	source, err := prepareUploadSource(path, "text/custom", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()

	if source.path != path {
		t.Fatalf("source path = %q, want %q", source.path, path)
	}
	if source.metadata.filename != "cat.txt" {
		t.Fatalf("filename = %q, want cat.txt", source.metadata.filename)
	}
	if source.metadata.contentType != "text/custom" {
		t.Fatalf("content type = %q, want text/custom", source.metadata.contentType)
	}
	if source.metadata.byteSize != int64(len(content)) {
		t.Fatalf("byte size = %d, want %d", source.metadata.byteSize, len(content))
	}
	if source.metadata.checksum != checksum(content) {
		t.Fatalf("checksum = %q, want %q", source.metadata.checksum, checksum(content))
	}
}

func TestPrepareUploadSourceURL(t *testing.T) {
	content := []byte("hello from url")
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header: http.Header{
					"Content-Disposition": {`attachment; filename="remote-cat.jpg"`},
					"Content-Type":        {"image/jpeg; charset=utf-8"},
				},
				Body:          io.NopCloser(bytes.NewReader(content)),
				ContentLength: int64(len(content)),
				Request:       req,
			}, nil
		}),
	}

	source, err := prepareUploadSource("https://example.com/ignored-name.png", "", client)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(source.path); err != nil {
		t.Fatalf("expected downloaded temp file to exist: %v", err)
	}

	if source.metadata.filename != "remote-cat.jpg" {
		t.Fatalf("filename = %q, want remote-cat.jpg", source.metadata.filename)
	}
	if source.metadata.contentType != "image/jpeg" {
		t.Fatalf("content type = %q, want image/jpeg", source.metadata.contentType)
	}
	if source.metadata.byteSize != int64(len(content)) {
		t.Fatalf("byte size = %d, want %d", source.metadata.byteSize, len(content))
	}
	if source.metadata.checksum != checksum(content) {
		t.Fatalf("checksum = %q, want %q", source.metadata.checksum, checksum(content))
	}

	if err := source.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(source.path); !os.IsNotExist(err) {
		t.Fatalf("expected temp file to be removed, stat err = %v", err)
	}
}

func checksum(content []byte) string {
	sum := md5.Sum(content)
	return base64.StdEncoding.EncodeToString(sum[:])
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
