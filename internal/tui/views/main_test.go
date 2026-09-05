package views

import (
	"os"
	"testing"
)

// TestMain initializes the style tables once for the whole package.
//
// The styles are package-level variables that stay at their zero value until
// InitStyles is called, and the zero value has no border and no padding. A
// handful of tests called InitStyles themselves, which left every other test
// in the package measuring whichever state the run order happened to produce.
// Under -shuffle=on that is a coin flip, and layout measured without the
// panel border and padding is short by six cells.
func TestMain(m *testing.M) {
	InitStyles()
	os.Exit(m.Run())
}
