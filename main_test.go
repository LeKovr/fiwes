package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	// Save original args
	a1 := os.Args[1]
	a2 := os.Args[2]

	tests := []struct {
		name string
		code int
		arg1 string
		arg2 string
	}{
		{"Help", 3, "-h", ""},
		{"UnknownFlag", 2, "-0", ""},
		{"UnknownPort", 1, "--http_addr", ":xx"},
	}
	for _, tt := range tests {
		os.Args[1] = tt.arg1
		os.Args[2] = tt.arg2
		var c int
		run(func(code int) { c = code })
		assert.Equal(t, tt.code, c, tt.name)
	}

	// Restore original args
	os.Args[1] = a1
	os.Args[2] = a2
}
