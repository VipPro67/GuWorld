package models

import (
	"database/sql"
	"time"

	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

type PlayerPet struct {
	ID                string    `json:"id"`
	PlayerID          string    `json:"player_id"`
	MonsterTemplateID string    `json:"monster_template_id"`
	Nickname          string    `json:"nickname"`
	Rank              int       `json:"rank"`
	SubRank           int       `json:"sub_rank"`
	HP                float64   `json:"hp"`
	MaxHP             float64   `json:"max_hp"`
	BaseAttack        float64   `json:"base_attack"`
	BaseDefense       float64   `json:"base_defense"`
	Speed             float64   `json:"speed"`
	Count             int       `json:"count"`
	Hunger            float64   `json:"hunger"`
	MaxHunger         float64   `json:"max_hunger"`
	PetHungerPerHour  float64   `json:"pet_hunger_per_hour"`
	State             string    `json:"state"`          // ACTIVE, SLEEPING
	FormationSlot     int       `json:"formation_slot"` // 0 = kho, 1..N = đội hình
	CapturedAt        time.Time `json:"captured_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func LoadPlayerPets(db *sql.DB, playerID string) ([]PlayerPet, error) {
	rows, err := db.Query(`
		SELECT id, player_id, monster_template_id, nickname, rank, sub_rank,
		       hp, max_hp, base_attack, base_defense, speed, count,
		       hunger, max_hunger, pet_hunger_per_hour, state, formation_slot,
		       captured_at, updated_at
		FROM player_pets
		WHERE player_id = $1
		ORDER BY formation_slot ASC, captured_at ASC
	`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pets []PlayerPet
	for rows.Next() {
		var p PlayerPet
		if err := rows.Scan(
			&p.ID, &p.PlayerID, &p.MonsterTemplateID, &p.Nickname,
			&p.Rank, &p.SubRank, &p.HP, &p.MaxHP, &p.BaseAttack, &p.BaseDefense, &p.Speed, &p.Count,
			&p.Hunger, &p.MaxHunger, &p.PetHungerPerHour, &p.State, &p.FormationSlot,
			&p.CapturedAt, &p.UpdatedAt,
		); err != nil {
			logger.S.Warnf("failed to scan PlayerPet: %v", err)
			continue
		}
		pets = append(pets, p)
	}
	return pets, rows.Err()
}

func LoadFormationPets(db *sql.DB, playerID string) ([]PlayerPet, error) {
	rows, err := db.Query(`
		SELECT id, player_id, monster_template_id, nickname, rank, sub_rank,
		       hp, max_hp, base_attack, base_defense, speed, count,
		       hunger, max_hunger, pet_hunger_per_hour, state, formation_slot,
		       captured_at, updated_at
		FROM player_pets
		WHERE player_id = $1 AND formation_slot > 0 AND state = 'ACTIVE' AND hunger > 0
		ORDER BY formation_slot ASC
	`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pets []PlayerPet
	for rows.Next() {
		var p PlayerPet
		if err := rows.Scan(
			&p.ID, &p.PlayerID, &p.MonsterTemplateID, &p.Nickname,
			&p.Rank, &p.SubRank, &p.HP, &p.MaxHP, &p.BaseAttack, &p.BaseDefense, &p.Speed, &p.Count,
			&p.Hunger, &p.MaxHunger, &p.PetHungerPerHour, &p.State, &p.FormationSlot,
			&p.CapturedAt, &p.UpdatedAt,
		); err != nil {
			logger.S.Warnf("failed to scan formation pet: %v", err)
			continue
		}
		pets = append(pets, p)
	}
	return pets, rows.Err()
}
