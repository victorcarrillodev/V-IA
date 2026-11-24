package main

import "testing"

func TestHelpYVersion(t *testing.T) {
	for _, a := range []string{"help", "--help", "-h", "version", "--version", "-v"} {
		if c := run([]string{a}); c != 0 {
			t.Errorf("%s devolvió %d", a, c)
		}
	}
	if c := run([]string{"nope"}); c != 2 {
		t.Errorf("comando desconocido debe devolver 2, no %d", c)
	}
}
