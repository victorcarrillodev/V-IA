package adapters

import "testing"

func TestParseMajor(t *testing.T) {
	for in, want := range map[string]int{"1.19.6\n": 1, "opencode v2.0.19": 2} {
		if n, ok := ParseMajor(in); !ok || n != want {
			t.Errorf("%q → %d,%v", in, n, ok)
		}
	}
	if _, ok := ParseMajor("sin versión"); ok {
		t.Error("texto sin versión no debe parsear")
	}
}
