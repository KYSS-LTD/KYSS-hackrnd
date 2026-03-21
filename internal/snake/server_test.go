package snake

import "testing"

func TestParseInputSupportsRussianLayout(t *testing.T) {
	cases := map[string]string{
		"ц": "ц",
		"ф": "ф",
		"ы": "ы",
		"в": "в",
		"й": "й",
	}
	for in, want := range cases {
		if got := parseInput([]byte(in)); got != want {
			t.Fatalf("parseInput(%q) = %q, want %q", in, got, want)
		}
	}
}
