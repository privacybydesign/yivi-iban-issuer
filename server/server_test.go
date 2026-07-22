package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestIsAllowedLanguage covers both the happy path (accepted language codes)
// and the rejection path (arbitrary / malicious values that must not reach the
// CM iDEAL return-URL interpolation).
func TestIsAllowedLanguage(t *testing.T) {
	allowed := []string{"nl", "en", "de"}
	for _, lang := range allowed {
		if !isAllowedLanguage(lang) {
			t.Errorf("expected language %q to be allowed", lang)
		}
	}

	rejected := []string{
		"",                     // empty
		"fr",                   // unsupported language
		"NL",                   // wrong case
		"nl ",                  // trailing space
		"../../evil",           // path traversal segment
		"nl/../evil",           // embedded traversal
		"https://evil.example", // full URL
		"%2e%2e%2fevil",        // percent-encoded traversal
		"nl%00",                // null byte
		"nl?redirect=evil.com", // extra query params
	}
	for _, lang := range rejected {
		if isAllowedLanguage(lang) {
			t.Errorf("expected language %q to be rejected", lang)
		}
	}
}

// stubIbanChecker records the language it was called with so the happy-path
// test can assert only allowlisted values reach StartIbanCheck.
type stubIbanChecker struct {
	calledWithLanguage string
	called             bool
}

func (s *stubIbanChecker) StartIbanCheck(entranceCode string, language string) (*IdealTransaction, error) {
	s.called = true
	s.calledWithLanguage = language
	return &IdealTransaction{
		TransactionID:           "txn-1",
		MerchantReference:       "ref-1",
		IssuerAuthenticationURL: "https://issuer.example/auth",
	}, nil
}

func (s *stubIbanChecker) GetStatus(merchantRef MerchantReference, transactionId TransactonId) (*TransactionStatus, error) {
	return nil, nil
}

func newTestState(checker IbanChecker) *ServerState {
	return &ServerState{
		ibanChecker:  checker,
		tokenStorage: NewInMemoryTokenStorage(),
	}
}

// TestHandleIBANCheckRejectsInvalidLanguage verifies that an unknown language
// value is rejected with 400 and never reaches StartIbanCheck.
func TestHandleIBANCheckRejectsInvalidLanguage(t *testing.T) {
	badValues := []string{"", "fr", "NL", "../../evil", "https://evil.example", "nl%00"}
	for _, lang := range badValues {
		checker := &stubIbanChecker{}
		state := newTestState(checker)

		body := `{"language":"` + lang + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/ibancheck", strings.NewReader(body))
		rec := httptest.NewRecorder()

		handleIBANCheck(state, rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("language %q: expected status %d, got %d", lang, http.StatusBadRequest, rec.Code)
		}
		if checker.called {
			t.Errorf("language %q: StartIbanCheck must not be called for a rejected language", lang)
		}
	}
}

// TestHandleIBANCheckAcceptsAllowedLanguage verifies allowlisted values pass
// validation and reach StartIbanCheck unchanged.
func TestHandleIBANCheckAcceptsAllowedLanguage(t *testing.T) {
	for _, lang := range []string{"nl", "en", "de"} {
		checker := &stubIbanChecker{}
		state := newTestState(checker)

		body := `{"language":"` + lang + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/ibancheck", strings.NewReader(body))
		rec := httptest.NewRecorder()

		handleIBANCheck(state, rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("language %q: expected status %d, got %d", lang, http.StatusOK, rec.Code)
		}
		if !checker.called {
			t.Errorf("language %q: expected StartIbanCheck to be called", lang)
		}
		if checker.calledWithLanguage != lang {
			t.Errorf("language %q: StartIbanCheck called with %q", lang, checker.calledWithLanguage)
		}
	}
}

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

// TestRespondWithErrWritesResponse verifies the rejection path: respondWithErr
// must write the given status code and body to the response instead of calling
// os.Exit(1) (which log.Error.Fatalf used to do, killing the whole server on a
// single failed request).
func TestRespondWithErrWritesResponse(t *testing.T) {
	rec := httptest.NewRecorder()

	respondWithErr(rec, http.StatusBadRequest, ErrorCannotValidateToken, "validation failed", errors.New("boom"))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if got := rec.Body.String(); got != ErrorCannotValidateToken {
		t.Fatalf("expected body %q, got %q", ErrorCannotValidateToken, got)
	}
}

// TestRespondWithErrDoesNotExit is a regression guard for the fix: the handler
// helper must return normally so that the goroutine serving the request (and the
// server as a whole) survives an error. If respondWithErr still called
// log.Error.Fatalf, the test binary would exit before this function returned and
// the test would be reported as failed by `go test`.
func TestRespondWithErrDoesNotExit(t *testing.T) {
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		respondWithErr(rec, http.StatusInternalServerError, "error:internal", "unexpected", errors.New("kaboom"))
		close(done)
	}()

	<-done

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

// TestRespondWithErrDifferentCodes exercises the happy path of the helper across
// a couple of status codes to confirm the code/body pair is written verbatim.
func TestRespondWithErrDifferentCodes(t *testing.T) {
	cases := []struct {
		name string
		code int
		body string
	}{
		{"rate limit", http.StatusTooManyRequests, ErrorRateLimit},
		{"phone format", http.StatusBadRequest, ErrorPhoneNumberFormat},
		{"malformed address", http.StatusBadRequest, ErrorAddressMalformed},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			respondWithErr(rec, tc.code, tc.body, "log message", errors.New("cause"))

			if rec.Code != tc.code {
				t.Errorf("expected status %d, got %d", tc.code, rec.Code)
			}
			if got := rec.Body.String(); got != tc.body {
				t.Errorf("expected body %q, got %q", tc.body, got)
			}
		})
	}
}
