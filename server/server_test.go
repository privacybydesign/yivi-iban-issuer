package main

import "testing"

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
