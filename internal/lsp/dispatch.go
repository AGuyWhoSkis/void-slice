package lsp

import (
	"encoding/json"
	"io"
	"os"

	"void-slice/internal/lint"
)

type rpcMessage struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method"`
	Params  json.RawMessage  `json:"params,omitempty"`
}

type rpcSuccess struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id"`
	Result  any              `json:"result"`
}

type rpcFailure struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id"`
	Error   *rpcError        `json:"error"`
}

type rpcNotification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

const jsonRPCVersion = "2.0"

func (s *Server) dispatch(w io.Writer, raw []byte) error {
	var msg rpcMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil
	}

	isRequest := msg.ID != nil

	switch msg.Method {
	case "initialize":
		return s.handleInitialize(w, msg)
	case "initialized":
		return nil
	case "shutdown":
		return s.handleShutdown(w, msg)
	case "exit":
		s.handleExit()
		return nil
	case "textDocument/didOpen":
		return s.handleDidOpen(w, msg)
	case "textDocument/didChange":
		return s.handleDidChange(w, msg)
	case "textDocument/didClose":
		return s.handleDidClose(w, msg)
	default:
		if isRequest {
			return writeMsg(w, rpcFailure{
				JSONRPC: jsonRPCVersion,
				ID:      msg.ID,
				Error:   &rpcError{Code: -32601, Message: "method not found: " + msg.Method},
			})
		}
		return nil
	}
}

func (s *Server) handleInitialize(w io.Writer, msg rpcMessage) error {
	return writeMsg(w, rpcSuccess{
		JSONRPC: jsonRPCVersion,
		ID:      msg.ID,
		Result:  InitializeResult{Capabilities: ServerCapabilities{TextDocumentSync: 1}},
	})
}

func (s *Server) handleShutdown(w io.Writer, msg rpcMessage) error {
	s.shutdownRequested = true
	return writeMsg(w, rpcSuccess{
		JSONRPC: jsonRPCVersion,
		ID:      msg.ID,
		Result:  nil,
	})
}

func (s *Server) handleExit() {
	if s.shutdownRequested {
		os.Exit(0)
	}
	os.Exit(1)
}

func (s *Server) handleDidOpen(w io.Writer, msg rpcMessage) error {
	var p DidOpenTextDocumentParams
	if err := json.Unmarshal(msg.Params, &p); err != nil {
		return nil
	}
	src := []byte(p.TextDocument.Text)
	s.docs[p.TextDocument.URI] = src
	return s.publishDiagnostics(w, p.TextDocument.URI, src)
}

func (s *Server) handleDidChange(w io.Writer, msg rpcMessage) error {
	var p DidChangeTextDocumentParams
	if err := json.Unmarshal(msg.Params, &p); err != nil {
		return nil
	}
	if len(p.ContentChanges) == 0 {
		return nil
	}
	src := []byte(p.ContentChanges[len(p.ContentChanges)-1].Text)
	s.docs[p.TextDocument.URI] = src
	return s.publishDiagnostics(w, p.TextDocument.URI, src)
}

func (s *Server) handleDidClose(w io.Writer, msg rpcMessage) error {
	var p DidCloseTextDocumentParams
	if err := json.Unmarshal(msg.Params, &p); err != nil {
		return nil
	}
	delete(s.docs, p.TextDocument.URI)
	return writeMsg(w, rpcNotification{
		JSONRPC: jsonRPCVersion,
		Method:  "textDocument/publishDiagnostics",
		Params:  PublishDiagnosticsParams{URI: p.TextDocument.URI, Diagnostics: []Diagnostic{}},
	})
}

func (s *Server) publishDiagnostics(w io.Writer, uri string, src []byte) error {
	lintDiags, err := lint.New().Lint(uri, src)
	if err != nil {
		lintDiags = nil
	}
	diags := convertDiagnostics(src, lintDiags)
	if diags == nil {
		diags = []Diagnostic{}
	}
	return writeMsg(w, rpcNotification{
		JSONRPC: jsonRPCVersion,
		Method:  "textDocument/publishDiagnostics",
		Params:  PublishDiagnosticsParams{URI: uri, Diagnostics: diags},
	})
}
