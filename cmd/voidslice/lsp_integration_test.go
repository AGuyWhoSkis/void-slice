package main_test

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type lspSession struct {
	t      *testing.T
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
}

func startLSP(t *testing.T) *lspSession {
	t.Helper()
	cmd := exec.Command(binaryPath, "lsp")
	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Start())
	return &lspSession{
		t:      t,
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReader(stdout),
	}
}

func (s *lspSession) send(v any) {
	s.t.Helper()
	body, err := json.Marshal(v)
	require.NoError(s.t, err)
	_, err = fmt.Fprintf(s.stdin, "Content-Length: %d\r\n\r\n", len(body))
	require.NoError(s.t, err)
	_, err = s.stdin.Write(body)
	require.NoError(s.t, err)
}

func (s *lspSession) recv() map[string]json.RawMessage {
	s.t.Helper()
	contentLength := -1
	for {
		line, err := s.stdout.ReadString('\n')
		require.NoError(s.t, err)
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if colon := strings.IndexByte(line, ':'); colon > 0 {
			name := strings.TrimSpace(line[:colon])
			val := strings.TrimSpace(line[colon+1:])
			if strings.EqualFold(name, "Content-Length") {
				contentLength, err = strconv.Atoi(val)
				require.NoError(s.t, err)
			}
		}
	}
	require.GreaterOrEqual(s.t, contentLength, 0, "missing Content-Length")
	body := make([]byte, contentLength)
	_, err := io.ReadFull(s.stdout, body)
	require.NoError(s.t, err)
	var m map[string]json.RawMessage
	require.NoError(s.t, json.Unmarshal(body, &m))
	return m
}

func (s *lspSession) waitExit(t *testing.T) int {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- s.cmd.Wait() }()
	select {
	case err := <-done:
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode()
		}
		if err != nil {
			t.Fatalf("cmd.Wait: %v", err)
		}
		return 0
	case <-time.After(5 * time.Second):
		_ = s.cmd.Process.Kill()
		t.Fatal("timeout waiting for lsp process to exit")
		return -1
	}
}

func TestLSP_FullSession(t *testing.T) {
	sess := startLSP(t)

	// initialize
	sess.send(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]any{"capabilities": map[string]any{}},
	})
	resp := sess.recv()
	var initResult struct {
		Capabilities struct {
			TextDocumentSync int `json:"textDocumentSync"`
		} `json:"capabilities"`
	}
	require.NoError(t, json.Unmarshal(resp["result"], &initResult))
	assert.Equal(t, 1, initResult.Capabilities.TextDocumentSync)

	// initialized (notification, no response expected)
	sess.send(map[string]any{
		"jsonrpc": "2.0",
		"method":  "initialized",
		"params":  map[string]any{},
	})

	// didOpen with broken fixture
	brokenSrc, err := os.ReadFile("../../testdata/broken/missing-semicolon.decl")
	require.NoError(t, err)
	sess.send(map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri":        "file:///test.decl",
				"languageId": "voidslice-decl",
				"version":    1,
				"text":       string(brokenSrc),
			},
		},
	})

	notif := sess.recv()
	assert.Equal(t, `"textDocument/publishDiagnostics"`, string(notif["method"]))
	var pubParams struct {
		URI         string `json:"uri"`
		Diagnostics []struct {
			Code     string `json:"code"`
			Severity int    `json:"severity"`
		} `json:"diagnostics"`
	}
	require.NoError(t, json.Unmarshal(notif["params"], &pubParams))
	assert.Equal(t, "file:///test.decl", pubParams.URI)
	require.NotEmpty(t, pubParams.Diagnostics)
	codes := make([]string, 0, len(pubParams.Diagnostics))
	for _, d := range pubParams.Diagnostics {
		codes = append(codes, d.Code)
	}
	assert.Contains(t, codes, "PARSE_EXPECTED_SEMICOLON")

	// didChange with valid content -> diagnostics: []
	clean := "Version 1\ncomponent {\n\tcpntTest myTest {\n\t\tedit = {\n\t\t\tm_val = \"hello\";\n\t\t}\n\t}\n}\n"
	sess.send(map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didChange",
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri":     "file:///test.decl",
				"version": 2,
			},
			"contentChanges": []map[string]any{
				{"text": clean},
			},
		},
	})

	notif = sess.recv()
	var rawParams struct {
		Diagnostics json.RawMessage `json:"diagnostics"`
	}
	require.NoError(t, json.Unmarshal(notif["params"], &rawParams))
	assert.Equal(t, "[]", string(rawParams.Diagnostics))

	// shutdown
	sess.send(map[string]any{
		"jsonrpc": "2.0",
		"id":      99,
		"method":  "shutdown",
	})
	resp = sess.recv()
	assert.Equal(t, "null", string(resp["result"]))

	// exit
	sess.send(map[string]any{
		"jsonrpc": "2.0",
		"method":  "exit",
	})
	require.NoError(t, sess.stdin.Close())

	assert.Equal(t, 0, sess.waitExit(t))
}

func TestLSP_ExitWithoutShutdown(t *testing.T) {
	sess := startLSP(t)

	sess.send(map[string]any{
		"jsonrpc": "2.0",
		"method":  "exit",
	})
	require.NoError(t, sess.stdin.Close())

	assert.Equal(t, 1, sess.waitExit(t))
}
