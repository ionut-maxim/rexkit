package rexstatus

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestReport(t *testing.T) {
	got := Report("app", Working, "line\nwith\x1b[31m controls")
	msg := base64.StdEncoding.EncodeToString([]byte("line with [31m controls"))
	if got != "\x1b]7501;state=working:app=app:msg="+msg+"\x1b\\" {
		t.Fatalf("report = %q", got)
	}
	if strings.Contains(Report("app", Idle, ""), "msg=") {
		t.Fatal("no message, no msg field")
	}
}
