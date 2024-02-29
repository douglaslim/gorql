package gorql

import (
	"strings"
	"testing"
)

func FuzzParse(f *testing.F) {
	f.Fuzz(func(t *testing.T, a string) {
		p, err := NewParser(nil)
		if err != nil {
			t.Fatalf("New parser error :%s", err)
		}
		_, _ = p.Parse(strings.NewReader(a))
	})
}
