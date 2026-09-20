package handlers

import (
	"database/sql"
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

type tradeEventPayload struct {
	Trade any `json:"trade"`
}

func notifyTradeParticipants(playerIDs ...uuid.UUID) {
	if len(playerIDs) == 0 {
		return
	}
	// Load trade once for the first participant
	trade, err := LoadActiveTrade(playerIDs[0])
	if err != nil {
		return
	}
	// When trade is nil (completed/not found), still notify both players
	// with {trade: null} so their UI can close the trade modal.
	payload := tradeEventPayload{Trade: trade}
	for _, playerID := range playerIDs {
		notifyTrade(playerID, payload)
	}
}

func notifyTrade(playerID uuid.UUID, payload tradeEventPayload) {
	_ = cache.PublishEventQuick("trade:"+playerID.String(), payload)
}

func getTradeParticipants(tradeID string) (uuid.UUID, uuid.UUID, error) {
	var senderID, receiverID uuid.UUID
	err := db.DB.QueryRow(`SELECT sender_id, receiver_id FROM trades WHERE id = $1`, tradeID).Scan(&senderID, &receiverID)
	return senderID, receiverID, err
}

func LoadActiveTrade(authID uuid.UUID) (any, error) {
	var id, status string
	var senderID, receiverID uuid.UUID
	var senderOfferBytes, receiverOfferBytes []byte
	var senderLocked, receiverLocked, senderConfirmed, receiverConfirmed bool
	var senderUsername, receiverUsername string

	err := db.DB.QueryRow(`
		SELECT t.id, t.sender_id, t.receiver_id, t.sender_offer, t.receiver_offer,
		       t.sender_locked, t.receiver_locked, t.sender_confirmed, t.receiver_confirmed,
		       t.status, p1.username, p2.username
		FROM trades t
		JOIN players p1 ON t.sender_id = p1.id
		JOIN players p2 ON t.receiver_id = p2.id
		WHERE (t.sender_id = $1 OR t.receiver_id = $1)
		  AND t.status IN ('PENDING', 'ACCEPTED')
		LIMIT 1
	`, authID).Scan(&id, &senderID, &receiverID, &senderOfferBytes, &receiverOfferBytes,
		&senderLocked, &receiverLocked, &senderConfirmed, &receiverConfirmed, &status,
		&senderUsername, &receiverUsername)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	var senderOffer, receiverOffer TradeOffer
	_ = json.Unmarshal(senderOfferBytes, &senderOffer)
	_ = json.Unmarshal(receiverOfferBytes, &receiverOffer)

	var senderGuInv models.GuInventory
	var receiverGuInv models.GuInventory
	_ = db.DB.QueryRow(`SELECT `+db.GuInventoryAggregate("players.id")+` FROM players WHERE id = $1`, senderID).Scan(&senderGuInv)
	_ = db.DB.QueryRow(`SELECT `+db.GuInventoryAggregate("players.id")+` FROM players WHERE id = $1`, receiverID).Scan(&receiverGuInv)

	guDetails := make(map[string]string)
	guRanks := make(map[string]int)
	for _, g := range senderGuInv {
		guDetails[g.InstanceID] = g.GuID
		guRanks[g.InstanceID] = g.Rank
	}
	for _, g := range receiverGuInv {
		guDetails[g.InstanceID] = g.GuID
		guRanks[g.InstanceID] = g.Rank
	}

	return gin.H{
		"id":                 id,
		"sender_id":          senderID,
		"sender_username":    senderUsername,
		"receiver_id":        receiverID,
		"receiver_username":  receiverUsername,
		"sender_offer":       senderOffer,
		"receiver_offer":     receiverOffer,
		"sender_locked":      senderLocked,
		"receiver_locked":    receiverLocked,
		"sender_confirmed":   senderConfirmed,
		"receiver_confirmed": receiverConfirmed,
		"status":             status,
		"gu_details":         guDetails,
		"gu_ranks":           guRanks,
	}, nil
}
