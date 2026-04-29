package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func frame(t *testing.T, v any) string {
	t.Helper()
	body, err := json.Marshal(v)
	require.NoError(t, err)
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
}

func readFramedAll(t *testing.T, r io.Reader) []map[string]json.RawMessage {
	t.Helper()
	br := bufio.NewReader(r)
	var msgs []map[string]json.RawMessage
	for {
		body, err := readMsg(br)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return msgs
			}
			t.Fatalf("readMsg: %v", err)
		}
		var m map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(body, &m))
		msgs = append(msgs, m)
	}
}

func TestInitialize(t *testing.T) {
	id := json.RawMessage(`1`)
	in := frame(t, rpcMessage{
		JSONRPC: jsonRPCVersion,
		ID:      &id,
		Method:  "initialize",
		Params:  json.RawMessage(`{}`),
	})
	out := &bytes.Buffer{}
	err := New().Serve(strings.NewReader(in), out)
	require.NoError(t, err)

	msgs := readFramedAll(t, bytes.NewReader(out.Bytes()))
	require.Len(t, msgs, 1)

	var result InitializeResult
	require.NoError(t, json.Unmarshal(msgs[0]["result"], &result))
	assert.Equal(t, 1, result.Capabilities.TextDocumentSync)
}

func TestDidOpen_BrokenFixture_PublishesDiagnostics(t *testing.T) {
	src, err := os.ReadFile("../../testdata/broken/missing-semicolon.decl")
	require.NoError(t, err)

	params := DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        "file:///test.decl",
			LanguageID: "voidslice-decl",
			Version:    1,
			Text:       string(src),
		},
	}
	rawParams, _ := json.Marshal(params)
	in := frame(t, rpcMessage{
		JSONRPC: jsonRPCVersion,
		Method:  "textDocument/didOpen",
		Params:  rawParams,
	})

	out := &bytes.Buffer{}
	require.NoError(t, New().Serve(strings.NewReader(in), out))

	msgs := readFramedAll(t, bytes.NewReader(out.Bytes()))
	require.Len(t, msgs, 1)
	assert.Equal(t, `"textDocument/publishDiagnostics"`, string(msgs[0]["method"]))

	var p PublishDiagnosticsParams
	require.NoError(t, json.Unmarshal(msgs[0]["params"], &p))
	assert.Equal(t, "file:///test.decl", p.URI)
	require.NotEmpty(t, p.Diagnostics)

	codes := make([]string, 0, len(p.Diagnostics))
	for _, d := range p.Diagnostics {
		codes = append(codes, d.Code)
		assert.Equal(t, "voidslice", d.Source)
	}
	assert.Contains(t, codes, "PARSE_EXPECTED_SEMICOLON")
}

func TestDidChange_FixedContent_EmptyDiagnostics(t *testing.T) {
	clean := "Version 1\ncomponent {\n\tcpntTest myTest {\n\t\tedit = {\n\t\t\tm_val = \"hello\";\n\t\t}\n\t}\n}\n"
	params := DidChangeTextDocumentParams{
		TextDocument: VersionedTextDocumentIdentifier{URI: "file:///test.decl", Version: 2},
		ContentChanges: []TextDocumentContentChangeEvent{
			{Text: clean},
		},
	}
	rawParams, _ := json.Marshal(params)
	in := frame(t, rpcMessage{
		JSONRPC: jsonRPCVersion,
		Method:  "textDocument/didChange",
		Params:  rawParams,
	})

	out := &bytes.Buffer{}
	require.NoError(t, New().Serve(strings.NewReader(in), out))

	msgs := readFramedAll(t, bytes.NewReader(out.Bytes()))
	require.Len(t, msgs, 1)

	// Verify raw JSON form: must be [] not null
	var p struct {
		URI         string          `json:"uri"`
		Diagnostics json.RawMessage `json:"diagnostics"`
	}
	require.NoError(t, json.Unmarshal(msgs[0]["params"], &p))
	assert.Equal(t, "[]", string(p.Diagnostics))
}

func TestShutdown_AndUnknownNotification(t *testing.T) {
	id := json.RawMessage(`42`)
	shutdownFrame := frame(t, rpcMessage{
		JSONRPC: jsonRPCVersion,
		ID:      &id,
		Method:  "shutdown",
	})
	unknownNotif := frame(t, rpcMessage{
		JSONRPC: jsonRPCVersion,
		Method:  "$/unknown",
	})

	srv := New()
	out := &bytes.Buffer{}
	require.NoError(t, srv.Serve(strings.NewReader(shutdownFrame+unknownNotif), out))

	assert.True(t, srv.shutdownRequested)

	msgs := readFramedAll(t, bytes.NewReader(out.Bytes()))
	require.Len(t, msgs, 1, "unknown notification must produce no output")

	assert.Equal(t, "null", string(msgs[0]["result"]))
	assert.Equal(t, "42", string(msgs[0]["id"]))
}

func TestUnknownRequest_ReturnsMethodNotFound(t *testing.T) {
	id := json.RawMessage(`7`)
	in := frame(t, rpcMessage{
		JSONRPC: jsonRPCVersion,
		ID:      &id,
		Method:  "totally/unknown",
	})
	out := &bytes.Buffer{}
	require.NoError(t, New().Serve(strings.NewReader(in), out))

	msgs := readFramedAll(t, bytes.NewReader(out.Bytes()))
	require.Len(t, msgs, 1)
	var e struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(msgs[0]["error"], &e))
	assert.Equal(t, -32601, e.Code)
}
