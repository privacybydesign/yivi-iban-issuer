package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestRespondWithErrWritesResponse verifies the rejection path: respondWithErr
// must write the given status code and body to the response instead of calling
// os.Exit(1) (which log.Error.Fatalf used to do, killing the whole server on a
// single failed request). See advisory GHSA-cjjv-hc9q-8p95.
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
