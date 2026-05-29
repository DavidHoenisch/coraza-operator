package source

import "testing"

func TestParseBlocklist(t *testing.T) {
	content := []byte("# comment\n192.0.2.1\n\n198.51.100.0/24\n")
	got := ParseBlocklist(content)
	want := []string{"192.0.2.1", "198.51.100.0/24"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
