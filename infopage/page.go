package main

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"
)

// dataPlaceholder is replaced in index.html by the JSON of the navigation request, so that the page shows
// the headers of the request the browser made itself, and not those of a later fetch, which differ
// (Accept, Sec-Fetch-*, JA4H).
const dataPlaceholder = "__INFOPAGE_DATA__"

// pageCSP only allows the page's own files: the data block is JSON, never executed.
const pageCSP = "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"

// wantsHTML keeps the text output of whoami for curl and scripts: only a GET or HEAD that accepts text/html gets
// the page, unless ?format= says otherwise.
func wantsHTML(r *http.Request) bool {
	switch r.URL.Query().Get("format") {
	case "html":
		return true
	case "text":
		return false
	}

	return (r.Method == http.MethodGet || r.Method == http.MethodHead) && strings.Contains(r.Header.Get("Accept"), "text/html")
}

func pageHandler(w http.ResponseWriter, r *http.Request) {
	page, err := staticFS.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// json.Marshal escapes <, > and &, so a header value can't close the script block.
	data, err := json.Marshal(newData(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	page = bytes.Replace(page, []byte(dataPlaceholder), data, 1)

	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Cache-Control", "no-store")
	h.Set("Content-Security-Policy", pageCSP)
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "no-referrer")
	_, _ = w.Write(page)
}

func assetsHandler() http.HandlerFunc {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}

	files := http.FileServerFS(sub)

	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		files.ServeHTTP(w, r)
	}
}
