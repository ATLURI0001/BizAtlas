package coords

import "testing"

func TestAssertHttpUrl(t *testing.T) {
	if err := AssertHttpUrl("https://maps.google.com/?q=cafe"); err != nil {
		t.Fatalf("expected valid URL, got %v", err)
	}
	if err := AssertHttpUrl("file:///etc/passwd"); err == nil {
		t.Fatal("expected URL scheme sentinel violation")
	}
}
