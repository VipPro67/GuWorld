package worker

import (
	"reflect"
	"testing"
)

func TestPartitionPendingPayloadsPreservesAlignment(t *testing.T) {
	ids := []string{"first", "middle", "last", "missing-result"}
	payloads := []string{"", `{"id":"middle"}`, `{"id":"last"}`}

	valid, missing := partitionPendingPayloads(ids, payloads)
	if want := []string{`{"id":"middle"}`, `{"id":"last"}`}; !reflect.DeepEqual(valid, want) {
		t.Fatalf("valid payloads = %#v, want %#v", valid, want)
	}
	if want := []string{"first", "missing-result"}; !reflect.DeepEqual(missing, want) {
		t.Fatalf("missing IDs = %#v, want %#v", missing, want)
	}
}
