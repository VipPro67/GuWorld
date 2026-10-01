package db

import "testing"

func TestDatasetRegistryConsistency(t *testing.T) {
	seen := make(map[string]bool)
	listed := AvailableTables()
	datasets := Datasets()
	if len(listed) != len(datasets) {
		t.Fatalf("AvailableTables has %d entries, want %d UI datasets", len(listed), len(datasets))
	}

	for index, dataset := range datasets {
		if dataset.Key == "" || seen[dataset.Key] {
			t.Fatalf("dataset key must be non-empty and unique: %q", dataset.Key)
		}
		seen[dataset.Key] = true
		if !dataset.UISupported {
			t.Fatalf("dataset %q is in the UI registry without UI support", dataset.Key)
		}
		if listed[index] != dataset.Key {
			t.Fatalf("AvailableTables[%d] = %q, want %q", index, listed[index], dataset.Key)
		}
		if dataset.Import == nil && dataset.Export == nil {
			t.Fatalf("UI dataset %q has no implementation", dataset.Key)
		}
		if dataset.Import != nil {
			if dataset.Preflight == nil {
				t.Errorf("importable dataset %q does not declare preflight", dataset.Key)
			}
			if dataset.Permissions.Import == "" {
				t.Errorf("importable dataset %q does not declare import permission", dataset.Key)
			}
			if len(dataset.CacheInvalidation) == 0 || len(dataset.RegistryReload) == 0 {
				t.Errorf("importable dataset %q must declare cache and registry behavior (use none explicitly)", dataset.Key)
			}
			for _, behavior := range dataset.CacheInvalidation {
				if !IsKnownDatasetCacheBehavior(behavior) {
					t.Errorf("dataset %q declares unknown cache behavior %q", dataset.Key, behavior)
				}
			}
			for _, registry := range dataset.RegistryReload {
				if !IsKnownDatasetRegistryReload(registry) {
					t.Errorf("dataset %q declares unknown registry reload %q", dataset.Key, registry)
				}
			}
		}
		if dataset.Export != nil && dataset.Permissions.Export == "" {
			t.Errorf("exportable dataset %q does not declare export permission", dataset.Key)
		}
		if dataset.Permissions.Read == "" || dataset.Label == "" || dataset.CSVFile == "" {
			t.Errorf("dataset %q has incomplete UI metadata", dataset.Key)
		}
	}

	if _, ok := DatasetByKey("node_output"); ok {
		t.Fatal("unimplemented node_output dataset must not be advertised")
	}
	if _, ok := DatasetByKey("sect"); ok {
		t.Fatal("export-only orphan sect dataset must not be advertised")
	}
}

func TestDatasetInfosReflectImplementations(t *testing.T) {
	for _, info := range DatasetInfos() {
		dataset, ok := DatasetByKey(info.Key)
		if !ok {
			t.Fatalf("published dataset %q is not registered", info.Key)
		}
		if info.Capabilities.Import != (dataset.Import != nil) ||
			info.Capabilities.Export != (dataset.Export != nil) ||
			info.Capabilities.Preflight != (dataset.Preflight != nil) {
			t.Errorf("published capabilities for %q do not match implementations", info.Key)
		}
	}
}
