package views

import (
	"fmt"
	"testing"

	"github.com/jp2195/pyre/internal/models"
)

func gpUsersFixture(n int) []models.GlobalProtectUser {
	users := make([]models.GlobalProtectUser, n)
	for i := range users {
		users[i] = models.GlobalProtectUser{Username: fmt.Sprintf("user%02d", i)}
	}
	return users
}

func TestGPUsersModel_SetSize_ClampsNegativeCursor(t *testing.T) {
	m := NewGPUsersModel()
	m = m.SetUsers(gpUsersFixture(5), nil)
	m.list.Cursor = -1

	m = m.SetSize(80, 30)

	if m.list.Cursor != 0 {
		t.Errorf("Cursor = %d, want 0 after clamping negative cursor", m.list.Cursor)
	}
}

func TestGPUsersModel_SetSize_ScrollsOffsetDownToCursor(t *testing.T) {
	m := NewGPUsersModel()
	m = m.SetUsers(gpUsersFixture(20), nil)
	// Cursor above the scroll window: offset must come down to the cursor.
	m.list.Cursor = 5
	m.list.Offset = 10

	m = m.SetSize(80, 30)

	if m.list.Offset > m.list.Cursor {
		t.Errorf("Offset = %d, want <= Cursor (%d) so the cursor row is visible", m.list.Offset, m.list.Cursor)
	}
}
