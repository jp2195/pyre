package tui

import (
	"os"
	"testing"

	"github.com/jp2195/pyre/internal/tui/views"
)

// TestMain initializes both style tables once for the whole package, the way
// main does before starting the program. Without it the styles keep their
// zero value, which has no border and no padding, and anything measuring a
// rendered layout measures something the user never sees.
func TestMain(m *testing.M) {
	InitStyles()
	views.InitStyles()
	os.Exit(m.Run())
}
