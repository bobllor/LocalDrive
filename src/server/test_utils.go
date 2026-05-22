package server

import (
	"testing"

	"github.com/bobllor/assert"
)

// NewTestServer creates a new Server test instance.
func NewTestServer(t *testing.T) *Server {
	addr := ":8080"

	serv, err := NewServer(addr)
	assert.Nil(t, err)

	return serv
}
