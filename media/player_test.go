package media

import (
	"strings"
	"testing"

	"github.com/example/velvetcut/editor"
)

func TestEDLCommasAndSemicolons(t *testing.T) {
	p := editor.NewProject()
	p.Clips = []editor.Clip{{ID: "a", Path: "/tmp/cut,with;stuff.mp4", SourceStart: 1.25, SourceEnd: 2.75, SourceDuration: 4}}
	edl, err := EDL(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(edl, "# mpv EDL v0\n%") || !strings.Contains(edl, "%/tmp/cut,with;stuff.mp4,1.250000,1.500000\n") {
		t.Fatalf("bad EDL: %q", edl)
	}
}
