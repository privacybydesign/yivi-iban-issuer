package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSpaHandlerStatErrorReturnsGenericMessage verifies that when os.Stat fails
// with an error other than "not exist" (here: a path whose parent is a regular
// file, yielding ENOTDIR), the handler returns a generic 500 body and does NOT
// echo the internal path or the raw error string back to the client.
func TestSpaHandlerStatErrorReturnsGenericMessage(t *testing.T) {
	dir := t.TempDir()

	// Create a regular file so that requesting a path *below* it produces a
	// stat error that is neither IsNotExist nor a directory.
	filePart := "regular.txt"
	if err := os.WriteFile(filepath.Join(dir, filePart), []byte("x"), 0o600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	h := spaHandler{staticPath: dir, indexPath: "index.html"}

	// Requesting "/regular.txt/child" makes filepath.Join produce
	// "<dir>/regular.txt/child"; stat on that returns ENOTDIR (not IsNotExist).
	req := httptest.NewRequest(http.MethodGet, "/"+filePart+"/child", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}

	body := rr.Body.String()

	if strings.Contains(body, dir) {
		t.Errorf("response body leaks internal filesystem path %q: %q", dir, body)
	}
	if strings.Contains(body, "no such") || strings.Contains(body, "not a directory") {
		t.Errorf("response body leaks raw stat error: %q", body)
	}

	want := http.StatusText(http.StatusInternalServerError)
	if strings.TrimSpace(body) != want {
		t.Errorf("expected generic body %q, got %q", want, strings.TrimSpace(body))
	}
}

// TestSpaHandlerMissingFileServesIndex verifies the happy fallback path: a
// request for a non-existent file serves index.html.
func TestSpaHandlerMissingFileServesIndex(t *testing.T) {
	dir := t.TempDir()

	indexContents := "<!doctype html><title>spa</title>"
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(indexContents), 0o600); err != nil {
		t.Fatalf("failed to create index.html: %v", err)
	}

	h := spaHandler{staticPath: dir, indexPath: "index.html"}

	req := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if got := rr.Body.String(); got != indexContents {
		t.Errorf("expected index.html contents, got %q", got)
	}
}

// TestSpaHandlerDirectoryServesIndex verifies that requesting a directory
// serves index.html (the fi.IsDir path, which now runs after the error check).
func TestSpaHandlerDirectoryServesIndex(t *testing.T) {
	dir := t.TempDir()

	indexContents := "<!doctype html><title>spa</title>"
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(indexContents), 0o600); err != nil {
		t.Fatalf("failed to create index.html: %v", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	h := spaHandler{staticPath: dir, indexPath: "index.html"}

	req := httptest.NewRequest(http.MethodGet, "/assets", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if got := rr.Body.String(); got != indexContents {
		t.Errorf("expected index.html contents, got %q", got)
	}
}

// TestSpaHandlerExistingFileIsServed verifies an existing static file is served
// with its real contents.
func TestSpaHandlerExistingFileIsServed(t *testing.T) {
	dir := t.TempDir()

	contents := "body{}"
	if err := os.WriteFile(filepath.Join(dir, "app.css"), []byte(contents), 0o600); err != nil {
		t.Fatalf("failed to create static file: %v", err)
	}

	h := spaHandler{staticPath: dir, indexPath: "index.html"}

	req := httptest.NewRequest(http.MethodGet, "/app.css", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if got := rr.Body.String(); got != contents {
		t.Errorf("expected static file contents %q, got %q", contents, got)
	}
}
