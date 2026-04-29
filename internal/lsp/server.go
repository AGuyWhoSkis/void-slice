package lsp

import (
	"bufio"
	"errors"
	"io"
)

type Server struct {
	docs              map[string][]byte
	shutdownRequested bool
}

func New() *Server {
	return &Server{docs: make(map[string][]byte)}
}

func (s *Server) Serve(r io.Reader, w io.Writer) error {
	br := bufio.NewReader(r)
	for {
		body, err := readMsg(br)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return nil
			}
			return err
		}
		if err := s.dispatch(w, body); err != nil {
			return err
		}
	}
}
