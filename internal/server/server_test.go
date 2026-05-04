package server_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"void-slice/internal/server"
)

func newTestServer(t *testing.T, cfg server.Config) *httptest.Server {
	t.Helper()
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	srv := httptest.NewServer(server.New(cfg))
	t.Cleanup(srv.Close)
	return srv
}

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "broken", name)
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	return b
}

type lintResp struct {
	File        string `json:"file"`
	Diagnostics []struct {
		Line     int    `json:"line"`
		Col      int    `json:"col"`
		Severity string `json:"severity"`
		Code     string `json:"code"`
		Message  string `json:"message"`
	} `json:"diagnostics"`
}

func TestHealth(t *testing.T) {
	srv := newTestServer(t, server.Config{})

	resp, err := http.Get(srv.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	body, _ := io.ReadAll(resp.Body)
	assert.JSONEq(t, `{"status":"ok"}`, string(body))
}

func TestLint_OctetStream_Broken(t *testing.T) {
	srv := newTestServer(t, server.Config{})
	src := loadFixture(t, "index-oob.decl")

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/lint?filename=index-oob.decl", bytes.NewReader(src))
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var got lintResp
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
	assert.Equal(t, "index-oob.decl", got.File)
	assert.NotEmpty(t, got.Diagnostics)
}

func TestLint_Clean_NoDiagnostics(t *testing.T) {
	srv := newTestServer(t, server.Config{})
	src := []byte("Version 1\ncomponent {\n\tcpntTest myTest {\n\t\tedit = {\n\t\t\tm_val = \"hello\";\n\t\t}\n\t}\n}\n")

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/lint?filename=clean.decl", bytes.NewReader(src))
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var got lintResp
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
	assert.Equal(t, "clean.decl", got.File)
	assert.Empty(t, got.Diagnostics)
}

func TestLint_OversizedBody_413(t *testing.T) {
	srv := newTestServer(t, server.Config{MaxBodyBytes: 16})

	body := bytes.Repeat([]byte("a"), 64)
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/lint", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode)
}

func TestLint_RejectsImageContentType(t *testing.T) {
	srv := newTestServer(t, server.Config{})

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/lint", bytes.NewReader([]byte("\x89PNG")))
	req.Header.Set("Content-Type", "image/png")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnsupportedMediaType, resp.StatusCode)
}

func TestLint_Multipart_UsesPartFilename(t *testing.T) {
	srv := newTestServer(t, server.Config{})
	src := loadFixture(t, "index-oob.decl")

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("file", "index-oob.decl")
	require.NoError(t, err)
	_, err = part.Write(src)
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/lint", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var got lintResp
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
	assert.Equal(t, "index-oob.decl", got.File)
	assert.NotEmpty(t, got.Diagnostics)
}

func TestLint_Timeout_504(t *testing.T) {
	srv := newTestServer(t, server.Config{LintTimeout: 1 * time.Nanosecond})
	src := loadFixture(t, "index-oob.decl")

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/lint", bytes.NewReader(src))
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusGatewayTimeout, resp.StatusCode)
}

func TestLint_Options_204_WithCORS(t *testing.T) {
	srv := newTestServer(t, server.Config{AllowedOrigin: "https://example.com"})

	req, _ := http.NewRequest(http.MethodOptions, srv.URL+"/lint", nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	assert.Equal(t, "https://example.com", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Contains(t, resp.Header.Get("Access-Control-Allow-Methods"), "POST")
	assert.Contains(t, resp.Header.Get("Access-Control-Allow-Headers"), "Content-Type")
	assert.Contains(t, resp.Header.Values("Vary"), "Origin")
}

func TestLint_GET_405(t *testing.T) {
	srv := newTestServer(t, server.Config{})

	resp, err := http.Get(srv.URL + "/lint")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}

func TestLint_CORSHeaderOnSuccess(t *testing.T) {
	srv := newTestServer(t, server.Config{AllowedOrigin: "*"})

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/lint", strings.NewReader("Version 1\ncomponent {}\n"))
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
}
