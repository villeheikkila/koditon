package mcpserver

import "testing"

func TestPropertyIDFromResourceURI(t *testing.T) {
	t.Parallel()
	got, err := propertyIDFromResourceURI("koditon://property/frontdoor%3Aad%3A123", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "frontdoor:ad:123" {
		t.Fatalf("id = %q, want frontdoor:ad:123", got)
	}
	got, err = propertyIDFromResourceURI("koditon://property/frontdoor%3Aad%3A123/report", "/report")
	if err != nil {
		t.Fatal(err)
	}
	if got != "frontdoor:ad:123" {
		t.Fatalf("report id = %q, want frontdoor:ad:123", got)
	}
}
