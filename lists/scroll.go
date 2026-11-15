// Package lists scrolls lists the way a touch screen does: a swipe (the
// wheel) moves the page and leaves the selection alone, even out of view;
// a key that moves the selection brings it back into view; a tap selects,
// and a tap on the selected row opens it.
//
// Scroll is the state for a list an app draws itself. Rows is a ready
// list of components.Row, with a filter, to use instead of a Bubbles list.
package lists

import tea "charm.land/bubbletea/v2"

// WheelLines is how far one wheel or swipe event scrolls.
const WheelLines = 3

// Scroll is where a list is scrolled to. The zero value is at the top,
// following the selection.
type Scroll struct {
	offset int  // the first line shown
	free   bool // scrolled by the wheel: don't pull the selection into view
	limit  int  // the largest offset, as of the last Place
}

// Wheel scrolls by WheelLines, freely: the selection may leave the view.
func (s *Scroll) Wheel(msg tea.MouseWheelMsg) {
	s.free = true
	switch msg.Mouse().Button {
	case tea.MouseWheelUp:
		s.offset -= WheelLines
	case tea.MouseWheelDown:
		s.offset += WheelLines
	}
	// Several events can come between two draws.
	s.offset = max(0, min(s.offset, s.limit))
}

// Follow brings the selection back into view at the next Place. Call it
// when a key moves or acts on the selection; not for taps, which land on
// what's in view already.
func (s *Scroll) Follow() { s.free = false }

// Reset goes back to the top, following the selection.
func (s *Scroll) Reset() { *s = Scroll{} }

// Free says the list was scrolled away from the selection.
func (s *Scroll) Free() bool { return s.free }

// Offset is the first line shown, as of the last Place.
func (s *Scroll) Offset() int { return s.offset }

// Place fixes the offset for a draw and returns it. The selection starts
// at line top and is h lines tall; unless scrolled freely, it's brought
// into view along with the line above it (a group header, or a gap).
// total is the content's height in lines, height the view's: the offset
// never goes past the end.
func (s *Scroll) Place(top, h, height, total int) int {
	if !s.free {
		if top-1 < s.offset {
			s.offset = max(top-1, 0)
		}
		if top+h > s.offset+height {
			s.offset = top + h - height
		}
	}
	s.limit = max(total-height, 0)
	s.offset = max(0, min(s.offset, s.limit))
	return s.offset
}
