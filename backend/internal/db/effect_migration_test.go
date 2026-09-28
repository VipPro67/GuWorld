package db

import "testing"

func TestEffectHashCanonicalizesObjectKeys(t *testing.T) {
	a, err := effectHash([]byte(`{"schema_version":2,"blocks":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	b, err := effectHash([]byte(`{"blocks":[],"schema_version":2}`))
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("canonical hashes differ: %s != %s", a, b)
	}
}

func TestKeyArgsPreservesCompositeGuKey(t *testing.T) {
	args, err := keyArgs([]byte(`{"rank":3,"id":"GU_TEST"}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(args) != 2 || args[0] != "GU_TEST" || args[1] != float64(3) {
		t.Fatalf("unexpected key args: %#v", args)
	}
}
