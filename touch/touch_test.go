package touch

import "testing"

func TestHitCoversTallZones(t *testing.T) {
	zones := []Zone{{Y: 0, X0: 2, X1: 6, H: 3, Msg: "a"}, {Y: 5, X0: 0, X1: 4, Msg: "b"}}
	for _, c := range []struct {
		x, y int
		want any
	}{{2, 0, "a"}, {5, 2, "a"}, {6, 0, nil}, {1, 5, "b"}, {1, 6, nil}} {
		got, ok := Hit(zones, c.x, c.y)
		if (c.want == nil) == ok || (ok && got != c.want) {
			t.Errorf("Hit(%d,%d) = %v,%v", c.x, c.y, got, ok)
		}
	}
}

func TestKeyMsg(t *testing.T) {
	if KeyMsg("D").String() != "D" || KeyMsg("enter").String() != "enter" {
		t.Fatal("KeyMsg")
	}
}
