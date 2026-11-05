// Package rexstatus reports what a terminal app is doing with Rex's
// Program Status Protocol (OSC 7501), so Rex can show a spinner on the
// app's tab while it works and a mark when it's done. Terminals that
// don't know the protocol ignore it.
package rexstatus

import (
	"encoding/base64"
	"strings"
)

// State is what the app is doing.
type State string

// The states Rex knows.
const (
	Idle    State = "idle"
	Working State = "working"
	Done    State = "done"
	Error   State = "error"
)

// Report is the OSC 7501 sequence for the app's root record: its state
// and, optionally, one short line of text.
func Report(app string, state State, msg string) string {
	var b strings.Builder
	b.WriteString("\x1b]7501;state=" + string(state) + ":app=" + app)
	if msg = OneLine(msg); msg != "" {
		b.WriteString(":msg=" + base64.StdEncoding.EncodeToString([]byte(msg)))
	}
	b.WriteString("\x1b\\")
	return b.String()
}

// OneLine drops control characters (the protocol refuses them) and keeps
// the text short.
func OneLine(s string) string {
	s = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r < 0xa0) {
			return ' '
		}
		return r
	}, s)
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > 200 {
		s = string(r[:199]) + "…"
	}
	return s
}

// Clear removes every record; send it when the app quits.
const Clear = "\x1b]7501;state=clear\x1b\\"
