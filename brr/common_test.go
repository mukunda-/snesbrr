// snesbrr
// Copyright 2025 Mukunda Johnson (mukunda.com)
// Licensed under MIT

package brr

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Setup code here (runs before all tests)

	exitCode := m.Run() // Run all tests

	// Teardown code here (runs after all tests)

	os.Exit(exitCode)
}
