package main

import (
	"net/http"
	"net/http/httptest"
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
// value is rejected with 400 and never reaches StartIbanCheck. Critically, the
// rejection must NOT crash the process: respondWithErr currently calls
// log.Error.Fatalf (os.Exit), so this path is written inline to return a plain
// 400. If that regressed, the test binary would exit non-zero here.
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
