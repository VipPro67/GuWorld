package nav

import "testing"

func FuzzParseMapNeverPanics(f *testing.F) {
	f.Add([]byte(`{"type":"map","orientation":"orthogonal","width":2,"height":2,"tilewidth":32,"tileheight":32,"map_id":"m","map_version":"v","layers":[{"type":"tilelayer","name":"collision","data":[0,0,0,0]}]}`))
	f.Add([]byte(`{"type":"map","orientation":"isometric","width":0,"height":0,"layers":[]}`))
	f.Add([]byte(`{"properties":[{"name":"ruleset","value":"pvp"}]}`))
	f.Add([]byte(`null`))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = Parse(data)
	})
}
