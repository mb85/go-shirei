package urilist

import (
	"slices"
	"testing"
)

func TestPaths(t *testing.T) {
	list := "# dragged from Dolphin\r\n" +
		"file:///home/me/My%20Images/arch.iso\r\n" +
		"file://localhost/tmp/a.img\r\n" +
		"https://example.com/remote.iso\r\n" +
		"file://otherhost/share/b.img\r\n" +
		"\r\n" +
		"file:///tmp/%C3%BCber.img"
	want := []string{"/home/me/My Images/arch.iso", "/tmp/a.img", "/tmp/über.img"}
	if got := Paths(list); !slices.Equal(got, want) {
		t.Fatalf("Paths = %q, want %q", got, want)
	}
}
