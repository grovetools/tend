package sessions

import (
	"testing"

	"github.com/grovetools/core/tui/keymap"
)

// TestKeyMapAuditCoverage asserts every enabled binding in the sessions KeyMap
// appears in a section (so nothing is silently missing from help) and that no
// help label contradicts its keys. If this fails, the Sections() list or the
// Base disable list in newKeyMap is wrong — fix the keymap, not the test.
func TestKeyMapAuditCoverage(t *testing.T) {
	if gaps := keymap.AuditCoverage(newKeyMap(nil)); len(gaps) != 0 {
		for _, g := range gaps {
			t.Errorf("audit gap: field=%s kind=%s detail=%s", g.Field, g.Kind, g.Detail)
		}
	}
}
