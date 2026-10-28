// Package nav loads the server side of the shared Tiled map artifact. The same
// JSON file is shipped to the Phaser client; a golden test keeps both copies
// byte-identical so collision and visuals cannot drift.
package nav

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/realtime/sim"
)

const (
	MaxMapTiles   = 16_777_216
	MaxSpawnCount = 4096
	MaxMapID      = 80
)

var (
	ErrInvalidMap = errors.New("invalid realtime map")
)

// TiledMapJSON is a deliberately small Tiled-compatible subset. One collision
// cell equals one world unit; tilewidth is visual metadata for the renderer.
type TiledMapJSON struct {
	Type           string          `json:"type"`
	Orientation    string          `json:"orientation"`
	RenderOrder    string          `json:"renderorder"`
	Width          int             `json:"width"`
	Height         int             `json:"height"`
	TileWidth      int             `json:"tilewidth"`
	TileHeight     int             `json:"tileheight"`
	MapID          string          `json:"map_id"`
	MapVersion     string          `json:"map_version"`
	ContentVersion string          `json:"content_version"`
	Layers         []TiledLayer    `json:"layers"`
	Properties     []TiledProperty `json:"properties,omitempty"`
}

type TiledLayer struct {
	Type    string        `json:"type"`
	Name    string        `json:"name"`
	Width   int           `json:"width,omitempty"`
	Height  int           `json:"height,omitempty"`
	Data    []int         `json:"data,omitempty"`
	Objects []TiledObject `json:"objects,omitempty"`
}

type TiledObject struct {
	ID         int             `json:"id,omitempty"`
	Name       string          `json:"name,omitempty"`
	Type       string          `json:"type,omitempty"`
	X          float64         `json:"x"`
	Y          float64         `json:"y"`
	Width      float64         `json:"width,omitempty"`
	Height     float64         `json:"height,omitempty"`
	Point      bool            `json:"point,omitempty"`
	Properties []TiledProperty `json:"properties,omitempty"`
}

type TiledProperty struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}

// Map is the validated immutable map used by zones and spawn selection.
type Map struct {
	ID             string
	Version        string
	ContentVersion string
	Ruleset        string
	Width          int
	Height         int
	TileWidth      int
	TileHeight     int
	Spawns         []Spawn
	Portals        []Portal
	Hash           string
	LegacyOriginX  float64
	LegacyOriginY  float64
	LegacyScale    float64

	collision sim.TileCollision
}

type Spawn struct {
	Name      string
	Kind      string
	MonsterID string
	Rank      int
	X         float64
	Y         float64
}

// Portal is an authored zone handoff point. Triggering it is the gateway/zone
// transfer layer's responsibility; the map only validates metadata.
type Portal struct {
	Name             string
	TargetZoneID     string
	TargetInstanceID string
	X                float64
	Y                float64
	TargetX          float64
	TargetY          float64
}

func LoadFile(path string) (*Map, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

func Parse(data []byte) (*Map, error) {
	if len(data) == 0 || len(data) > 8*1024*1024 {
		return nil, fmt.Errorf("%w: file size", ErrInvalidMap)
	}
	var raw TiledMapJSON
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidMap, err)
	}
	return Build(raw)
}

func Build(raw TiledMapJSON) (*Map, error) {
	if raw.Type != "" && raw.Type != "map" {
		return nil, fmt.Errorf("%w: type must be map", ErrInvalidMap)
	}
	if raw.Orientation != "orthogonal" {
		return nil, fmt.Errorf("%w: only orthogonal maps are supported", ErrInvalidMap)
	}
	if raw.Width <= 0 || raw.Height <= 0 || raw.Width > MaxMapTiles/raw.Height {
		return nil, fmt.Errorf("%w: invalid dimensions", ErrInvalidMap)
	}
	if raw.TileWidth <= 0 || raw.TileHeight <= 0 {
		return nil, fmt.Errorf("%w: tile size must be positive", ErrInvalidMap)
	}
	if !validIdentifier(raw.MapID) || !validIdentifier(raw.MapVersion) {
		return nil, fmt.Errorf("%w: map_id and map_version are required", ErrInvalidMap)
	}

	var collision sim.TileCollision
	hasCollision := false
	var spawns []Spawn
	var portals []Portal
	for _, layer := range raw.Layers {
		switch layer.Type {
		case "tilelayer":
			if len(layer.Data) != 0 && len(layer.Data) != raw.Width*raw.Height {
				return nil, fmt.Errorf("%w: layer %q data must be %d cells", ErrInvalidMap, layer.Name, raw.Width*raw.Height)
			}
			switch layer.Name {
			case "collision":
				hasCollision = true
				var err error
				if len(layer.Data) != 0 {
					collision, err = sim.NewPackedCollision(raw.Width, raw.Height, layer.Data)
					if err != nil {
						return nil, fmt.Errorf("%w: collision cells must be 0 or 1", ErrInvalidMap)
					}
				} else {
					collision, err = sim.NewBorderCollision(raw.Width, raw.Height)
					if err != nil {
						return nil, fmt.Errorf("%w: %v", ErrInvalidMap, err)
					}
				}
			}
		case "objectgroup":
			if layer.Name == "portals" {
				for _, object := range layer.Objects {
					targetZone := propertyString(object.Properties, "target_zone_id")
					targetX, okX := propertyFloat(object.Properties, "target_x")
					targetY, okY := propertyFloat(object.Properties, "target_y")
					if !validIdentifier(targetZone) || !okX || !okY || !finite(targetX) || !finite(targetY) || targetX < 0 || targetY < 0 {
						return nil, fmt.Errorf("%w: portal %q needs target_zone_id/target_x/target_y", ErrInvalidMap, object.Name)
					}
					if object.X < 0 || object.Y < 0 || object.X >= float64(raw.Width) || object.Y >= float64(raw.Height) {
						return nil, fmt.Errorf("%w: portal %q is outside the map", ErrInvalidMap, object.Name)
					}
					if collision != nil && collision.SolidTile(int(math.Floor(object.X)), int(math.Floor(object.Y))) {
						return nil, fmt.Errorf("%w: portal %q is inside a solid tile", ErrInvalidMap, object.Name)
					}
					portals = append(portals, Portal{
						Name: object.Name, TargetZoneID: targetZone,
						TargetInstanceID: propertyString(object.Properties, "target_instance_id"),
						X:                object.X, Y: object.Y, TargetX: targetX, TargetY: targetY,
					})
				}
				continue
			}
			if layer.Name != "spawns" {
				continue
			}
			for _, object := range layer.Objects {
				if len(spawns) >= MaxSpawnCount {
					return nil, fmt.Errorf("%w: too many spawn points", ErrInvalidMap)
				}
				if !validIdentifier(object.Type) {
					return nil, fmt.Errorf("%w: spawn object needs a type", ErrInvalidMap)
				}
				if object.X < 0 || object.Y < 0 || object.X >= float64(raw.Width) || object.Y >= float64(raw.Height) {
					return nil, fmt.Errorf("%w: spawn %q is outside the map", ErrInvalidMap, object.Name)
				}
				if collision != nil && collision.SolidTile(int(math.Floor(object.X)), int(math.Floor(object.Y))) {
					return nil, fmt.Errorf("%w: spawn %q is inside a solid tile", ErrInvalidMap, object.Name)
				}
				monsterID := propertyString(object.Properties, "monster_id")
				if monsterID != "" && !validIdentifier(monsterID) {
					return nil, fmt.Errorf("%w: spawn %q monster_id is invalid", ErrInvalidMap, object.Name)
				}
				rankValue, _ := propertyFloat(object.Properties, "rank")
				rank := int(rankValue)
				if rank < 0 || rankValue != float64(rank) {
					return nil, fmt.Errorf("%w: spawn %q rank must be a non-negative integer", ErrInvalidMap, object.Name)
				}
				spawns = append(spawns, Spawn{Name: object.Name, Kind: object.Type, MonsterID: monsterID, Rank: rank, X: object.X, Y: object.Y})
			}
		}
	}
	if !hasCollision {
		return nil, fmt.Errorf("%w: collision layer is required", ErrInvalidMap)
	}
	ruleset := strings.ToLower(propertyString(raw.Properties, "ruleset"))
	legacyOriginX, _ := propertyFloat(raw.Properties, "legacy_origin_x")
	legacyOriginY, _ := propertyFloat(raw.Properties, "legacy_origin_y")
	legacyScale, hasLegacyScale := propertyFloat(raw.Properties, "legacy_units_per_world_unit")
	if !hasLegacyScale || legacyScale <= 0 || !finite(legacyScale) {
		legacyScale = 1
	}
	if ruleset == "" {
		ruleset = "pve"
	}
	if ruleset != "pve" && ruleset != "pvp" && ruleset != "safe" {
		return nil, fmt.Errorf("%w: ruleset must be pve/pvp/safe", ErrInvalidMap)
	}
	hash := hashMap(raw, collision)
	return &Map{
		ID: raw.MapID, Version: raw.MapVersion, ContentVersion: raw.ContentVersion, Ruleset: ruleset,
		Width: raw.Width, Height: raw.Height, TileWidth: raw.TileWidth, TileHeight: raw.TileHeight,
		Spawns: spawns, Portals: portals, Hash: hash, collision: collision,
		LegacyOriginX: legacyOriginX, LegacyOriginY: legacyOriginY, LegacyScale: legacyScale,
	}, nil
}

func (m *Map) Collision() sim.Collision {
	if m == nil {
		return nil
	}
	return m.collision
}

func (m *Map) IsSolid(x, y float64) bool {
	if m == nil || m.collision == nil || !finite(x) || !finite(y) {
		return true
	}
	return m.collision.Blocked(sim.Vec2{X: x, Y: y}, 0)
}

// SafeSpawn returns the preferred position when it is walkable, then the first
// authored spawn, then the first free tile. It keeps admission from creating a
// player entity inside a wall after legacy coordinates are converted.
func (m *Map) SafeSpawn(preferred sim.Vec2, radius float64) (sim.Vec2, bool) {
	if m == nil || m.collision == nil {
		return preferred, true
	}
	if !m.collision.Blocked(preferred, radius) {
		return preferred, true
	}
	for _, spawn := range m.Spawns {
		position := sim.Vec2{X: spawn.X, Y: spawn.Y}
		if !m.collision.Blocked(position, radius) {
			return position, true
		}
	}
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			position := sim.Vec2{X: float64(x) + 0.5, Y: float64(y) + 0.5}
			if !m.collision.Blocked(position, radius) {
				return position, true
			}
		}
	}
	return preferred, false
}

func hashMap(raw TiledMapJSON, collision sim.TileCollision) string {
	hash := sha256.New()
	_, _ = fmt.Fprintf(hash, "%s\x00%s\x00%d\x00%d\x00", raw.MapID, raw.MapVersion, raw.Width, raw.Height)
	var chunk [8192]byte
	used := 0
	for y := 0; y < raw.Height; y++ {
		for x := 0; x < raw.Width; x++ {
			if collision.SolidTile(x, y) {
				chunk[used] = 1
			} else {
				chunk[used] = 0
			}
			used++
			if used == len(chunk) {
				_, _ = hash.Write(chunk[:])
				used = 0
			}
		}
	}
	_, _ = hash.Write(chunk[:used])
	return hex.EncodeToString(hash.Sum(nil))
}

func validIdentifier(value string) bool {
	return value != "" && len(value) <= MaxMapID && strings.TrimSpace(value) == value
}

func propertyFloat(properties []TiledProperty, name string) (float64, bool) {
	for _, property := range properties {
		if property.Name != name {
			continue
		}
		switch value := property.Value.(type) {
		case float64:
			return value, true
		case json.Number:
			parsed, err := value.Float64()
			if err == nil {
				return parsed, true
			}
		}
	}
	return 0, false
}

func propertyString(properties []TiledProperty, name string) string {
	for _, property := range properties {
		if property.Name != name {
			continue
		}
		value, ok := property.Value.(string)
		if !ok {
			return ""
		}
		return strings.TrimSpace(value)
	}
	return ""
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
