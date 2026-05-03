package sourcejson

import "testing"

func TestCanonicalizeAndHashIgnoresObjectKeyOrder(t *testing.T) {
	t.Parallel()
	firstCanonical, firstHash, err := CanonicalizeAndHash([]byte(`{"b":2,"a":1}`))
	if err != nil {
		t.Fatalf("first hash failed: %v", err)
	}
	secondCanonical, secondHash, err := CanonicalizeAndHash([]byte(`{"a":1,"b":2}`))
	if err != nil {
		t.Fatalf("second hash failed: %v", err)
	}
	if firstHash != secondHash {
		t.Fatalf("hash mismatch: %s != %s", firstHash, secondHash)
	}
	if string(firstCanonical) != string(secondCanonical) {
		t.Fatalf("canonical mismatch: %s != %s", firstCanonical, secondCanonical)
	}
}

func TestCanonicalizeAndHashRejectsTrailingData(t *testing.T) {
	t.Parallel()
	_, _, err := CanonicalizeAndHash([]byte(`{"a":1}{"b":2}`))
	if err == nil {
		t.Fatal("expected trailing data error")
	}
}
