package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

type tradeEventPayload struct {
	Trade any `json:"trade"`
}

type tradeSubscription struct {
	playerID uuid.UUID
	ch       chan tradeEventPayload
}

var tradeEventHub = struct {
	sync.Mutex
	subs map[uuid.UUID]map[chan tradeEventPayload]struct{}
}{subs: map[uuid.UUID]map[chan tradeEventPayload]struct{}{}}

func SubscribeTradeEvents(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming unsupported"})
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache, no-transform")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Header().Set("Content-Encoding", "identity")

	ch := subscribeTrade(playerID)
	defer unsubscribeTrade(playerID, ch)

	if err := writeTradeEvent(c, playerID); err != nil {
		_ = writeSSE(c, "error", gin.H{"error": err.Error()})
		flusher.Flush()
		return
	}
	flusher.Flush()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case payload := <-ch:
			if err := writeSSE(c, "trade.updated", payload); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func subscribeTrade(playerID uuid.UUID) chan tradeEventPayload {
	ch := make(chan tradeEventPayload, 1)
	tradeEventHub.Lock()
	defer tradeEventHub.Unlock()
	if tradeEventHub.subs[playerID] == nil {
		tradeEventHub.subs[playerID] = map[chan tradeEventPayload]struct{}{}
	}
	tradeEventHub.subs[playerID][ch] = struct{}{}
	return ch
}

func unsubscribeTrade(playerID uuid.UUID, ch chan tradeEventPayload) {
	tradeEventHub.Lock()
	defer tradeEventHub.Unlock()
	delete(tradeEventHub.subs[playerID], ch)
	if len(tradeEventHub.subs[playerID]) == 0 {
		delete(tradeEventHub.subs, playerID)
	}
	close(ch)
}

func notifyTradeParticipants(playerIDs ...uuid.UUID) {
	for _, playerID := range playerIDs {
		trade, err := loadActiveTrade(playerID)
		if err != nil {
			continue
		}
		notifyTrade(playerID, tradeEventPayload{Trade: trade})
	}
}

func notifyTrade(playerID uuid.UUID, payload tradeEventPayload) {
	tradeEventHub.Lock()
	defer tradeEventHub.Unlock()
	for ch := range tradeEventHub.subs[playerID] {
		select {
		case ch <- payload:
		default:
			select {
			case <-ch:
			default:
			}
			ch <- payload
		}
	}
}

func writeTradeEvent(c *gin.Context, playerID uuid.UUID) error {
	trade, err := loadActiveTrade(playerID)
	if err != nil {
		return err
	}
	return writeSSE(c, "trade.updated", tradeEventPayload{Trade: trade})
}

func getTradeParticipants(tradeID string) (uuid.UUID, uuid.UUID, error) {
	var senderID, receiverID uuid.UUID
	err := db.DB.QueryRow(`SELECT sender_id, receiver_id FROM trades WHERE id = $1`, tradeID).Scan(&senderID, &receiverID)
	return senderID, receiverID, err
}

func loadActiveTrade(authID uuid.UUID) (any, error) {
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
	for _, g := range senderGuInv {
		guDetails[g.InstanceID] = g.GuID
	}
	for _, g := range receiverGuInv {
		guDetails[g.InstanceID] = g.GuID
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
	}, nil
}
