package server

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"void-slice/internal/lint"
	"void-slice/internal/report"
	"void-slice/internal/scan"
)

const (
	// 1 MiB matches Cloudflare Workers' realistic memory headroom for the
	// linter; see kanban/done/v2/T26-linter-resource-profile.md.
	defaultMaxBodyBytes = 1 << 20
	defaultLintTimeout  = 5 * time.Second
)

type Config struct {
	AllowedOrigin string
	MaxBodyBytes  int64
	LintTimeout   time.Duration
	Logger        *slog.Logger
}

func (c Config) withDefaults() Config {
	if c.AllowedOrigin == "" {
		c.AllowedOrigin = "*"
	}
	if c.MaxBodyBytes <= 0 {
		c.MaxBodyBytes = defaultMaxBodyBytes
	}
	if c.LintTimeout <= 0 {
		c.LintTimeout = defaultLintTimeout
	}
	if c.Logger == nil {
		c.Logger = slog.Default()
	}
	return c
}

func New(cfg Config) http.Handler {
	cfg = cfg.withDefaults()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	mux.Handle("/lint", &lintHandler{cfg: cfg})

	return logging(cfg.Logger, cors(cfg.AllowedOrigin, mux))
}

func ListenAndServe(addr string, cfg Config) error {
	return http.ListenAndServe(addr, New(cfg))
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, `{"status":"ok"}`)
}

type lintHandler struct {
	cfg Config
}

func (h *lintHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodOptions:
		w.WriteHeader(http.StatusNoContent)
		return
	case http.MethodPost:
		// fall through
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ct := r.Header.Get("Content-Type")
	if isMultipart(ct) {
		h.handleMultipart(w, r)
		return
	}
	if !isAllowedContentType(ct) {
		http.Error(w, "unsupported media type", http.StatusUnsupportedMediaType)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, h.cfg.MaxBodyBytes)
	src, err := io.ReadAll(r.Body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "could not read body", http.StatusBadRequest)
		return
	}

	filename := r.URL.Query().Get("filename")
	if filename == "" {
		filename = "input"
	}

	h.lintAndRespond(w, r.Context(), filename, src)
}

func (h *lintHandler) handleMultipart(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, h.cfg.MaxBodyBytes)
	if err := r.ParseMultipartForm(h.cfg.MaxBodyBytes); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "could not parse multipart body", http.StatusBadRequest)
		return
	}
	if r.MultipartForm == nil || len(r.MultipartForm.File) == 0 {
		http.Error(w, "missing file part", http.StatusBadRequest)
		return
	}

	var fileHeader *multipartFile
	for _, headers := range r.MultipartForm.File {
		if len(headers) > 0 {
			fileHeader = &multipartFile{name: headers[0].Filename, contentType: headers[0].Header.Get("Content-Type")}
			f, err := headers[0].Open()
			if err != nil {
				http.Error(w, "could not open uploaded file", http.StatusBadRequest)
				return
			}
			defer f.Close()
			fileHeader.body = f
			break
		}
	}
	if fileHeader == nil {
		http.Error(w, "missing file part", http.StatusBadRequest)
		return
	}
	if fileHeader.contentType != "" && !isAllowedContentType(fileHeader.contentType) {
		http.Error(w, "unsupported media type", http.StatusUnsupportedMediaType)
		return
	}

	src, err := io.ReadAll(fileHeader.body)
	if err != nil {
		http.Error(w, "could not read uploaded file", http.StatusBadRequest)
		return
	}

	filename := fileHeader.name
	if filename == "" {
		filename = r.URL.Query().Get("filename")
	}
	if filename == "" {
		filename = "input"
	}

	h.lintAndRespond(w, r.Context(), filename, src)
}

type multipartFile struct {
	name        string
	contentType string
	body        io.ReadCloser
}

type lintResult struct {
	body string
	err  error
}

func (h *lintHandler) lintAndRespond(w http.ResponseWriter, parent context.Context, filename string, src []byte) {
	ctx, cancel := context.WithTimeout(parent, h.cfg.LintTimeout)
	defer cancel()

	resultCh := make(chan lintResult, 1)
	go func() {
		diags, err := lint.New().Lint(filename, src)
		if err != nil {
			resultCh <- lintResult{err: err}
			return
		}
		scanDiags := make([]scan.Diagnostic, len(diags))
		for i, d := range diags {
			scanDiags[i] = scan.Diagnostic{Code: scan.DiagnosticCode(d.Code), Span: d.Span, Message: d.Message}
		}
		resultCh <- lintResult{body: report.RenderJSON(filename, src, scanDiags)}
	}()

	select {
	case <-ctx.Done():
		http.Error(w, "lint timed out", http.StatusGatewayTimeout)
	case res := <-resultCh:
		if res.err != nil {
			http.Error(w, "lint failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, res.body)
	}
}

func isMultipart(ct string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(ct)), "multipart/form-data")
}

func isAllowedContentType(ct string) bool {
	ct = strings.ToLower(strings.TrimSpace(ct))
	if ct == "" {
		return true
	}
	if i := strings.Index(ct, ";"); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}
	switch {
	case ct == "application/octet-stream":
		return true
	case strings.HasPrefix(ct, "text/"):
		return true
	case strings.HasPrefix(ct, "image/"),
		strings.HasPrefix(ct, "audio/"),
		strings.HasPrefix(ct, "video/"):
		return false
	case ct == "application/zip",
		ct == "application/gzip",
		ct == "application/x-gzip",
		ct == "application/x-tar",
		ct == "application/pdf":
		return false
	}
	return true
}

func cors(origin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Access-Control-Allow-Origin", origin)
		h.Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		h.Set("Access-Control-Allow-Headers", "Content-Type")
		if origin != "*" {
			h.Add("Vary", "Origin")
		}
		next.ServeHTTP(w, r)
	})
}

func logging(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		logger.Info("request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", rw.status),
			slog.Int64("duration_ms", time.Since(start).Milliseconds()),
			slog.Int64("bytes_in", r.ContentLength),
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if r.wroteHeader {
		return
	}
	r.wroteHeader = true
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.wroteHeader = true
	}
	return r.ResponseWriter.Write(b)
}
