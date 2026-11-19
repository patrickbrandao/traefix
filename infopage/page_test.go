package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInfopageHandlerFormats(t *testing.T) {
	testCases := []struct {
		desc   string
		target string
		accept string
		html   bool
	}{
		{desc: "browser", target: "/", accept: "text/html,application/xhtml+xml,*/*;q=0.8", html: true},
		{desc: "curl", target: "/", accept: "*/*"},
		{desc: "browser asking text", target: "/?format=text", accept: "text/html"},
		{desc: "curl asking html", target: "/?format=html", accept: "*/*", html: true},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, test.target, http.NoBody)
			req.Header.Set("Accept", test.accept)
			rw := httptest.NewRecorder()

			infopageHandler(rw, req)

			body := rw.Body.String()
			if test.html {
				if !strings.HasPrefix(rw.Header().Get("Content-Type"), "text/html") || !strings.Contains(body, `id="infopage-data"`) {
					t.Fatalf("expected the HTML page, got %q", body)
				}
				if strings.Contains(body, dataPlaceholder) {
					t.Fatal("placeholder not replaced")
				}
				return
			}
			if !strings.Contains(body, "Hostname:") || strings.Contains(body, "<html") {
				t.Fatalf("expected the text output, got %q", body)
			}
		})
	}
}

func TestPageHandlerEscapesHeaders(t *testing.T) {
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
	req.Header.Set("Accept", "text/html")
	req.Header.Set("X-Evil", "</script><script>alert(1)</script>")
	rw := httptest.NewRecorder()

	infopageHandler(rw, req)

	if strings.Contains(rw.Body.String(), "<script>alert(1)") {
		t.Fatal("header value not escaped")
	}
}

func TestAssetsHandler(t *testing.T) {
	h := assetsHandler()

	for path, code := range map[string]int{"/assets/app.js": http.StatusOK, "/assets/style.css": http.StatusOK, "/assets/": http.StatusNotFound, "/assets/nope.js": http.StatusNotFound} {
		rw := httptest.NewRecorder()
		h(rw, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, http.NoBody))
		if rw.Code != code {
			t.Errorf("%s: got %d, want %d", path, rw.Code, code)
		}
	}
}
