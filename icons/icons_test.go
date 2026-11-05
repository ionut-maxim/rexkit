package icons

import (
	"reflect"
	"testing"
)

// Every glyph has a Nerd Font and a Unicode version, so an app never draws
// an empty icon with either set.
func TestSetsAreComplete(t *testing.T) {
	for name, set := range map[string]Set{"Nerd": Nerd, "Unicode": Unicode} {
		v := reflect.ValueOf(set)
		for i := range v.NumField() {
			f := v.Field(i)
			if f.Kind() != reflect.String {
				continue
			}
			field := v.Type().Field(i).Name
			if field == "PillL" || field == "PillR" {
				continue // Unicode has no pill ends on purpose
			}
			if f.String() == "" {
				t.Errorf("%s.%s is empty", name, field)
			}
		}
	}
}
