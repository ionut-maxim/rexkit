package components

import "testing"

func TestSizeOf(t *testing.T) {
	for w, want := range map[int]Size{50: Compact, 63: Compact, 64: Regular, 83: Regular, 119: Regular, 120: Wide, 165: Wide} {
		if got := SizeOf(w); got != want {
			t.Errorf("SizeOf(%d) = %v, want %v", w, got, want)
		}
	}
}

func TestTallButtonsAreThreeRows(t *testing.T) {
	b := Btn("k", "label", nil, true)
	if lines := RenderSegs([]Seg{b}, true); len(lines) != 3 || lines[0] != "" || lines[2] != "" {
		t.Fatalf("tall row = %q", lines)
	}
	if lines := RenderSegs([]Seg{b}, false); len(lines) != 1 {
		t.Fatalf("row = %q", lines)
	}
}
