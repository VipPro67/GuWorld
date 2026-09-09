package ws

import (
	"encoding/json"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// StartRedisSubscribers spawns goroutines to listen to Redis Pub/Sub channels
// and forward messages to the WebSocket Hub.
func StartRedisSubscribers(hub *Hub) {
	go subscribePattern(hub, "player:*")
	go subscribePattern(hub, "chat:*")
	go subscribePattern(hub, "trade:*")
	go subscribePattern(hub, "world:pos")
	go subscribePattern(hub, "world:cell")
	go subscribePattern(hub, "combat:*")
	go subscribePattern(hub, "notif:*")
}

func subscribePattern(hub *Hub, pattern string) {
	for {
		if hub.ctx.Err() != nil {
			return // Hub is stopping
		}

		if cache.Client == nil {
			time.Sleep(5 * time.Second)
			continue
		}

		pubsub := cache.Client.PSubscribe(hub.ctx, pattern)
		// Wait for subscription to be created before moving on.
		_, err := pubsub.Receive(hub.ctx)
		if err != nil {
			logger.S.Warnf("Failed to subscribe to pattern %s: %v", pattern, err)
			pubsub.Close()
			time.Sleep(2 * time.Second) // backoff
			continue
		}

		logger.S.Infof("Subscribed to Redis pattern: %s", pattern)

		ch := pubsub.Channel()

		// Receive loop
		for msg := range ch {
			_, clientMsg := formatClientEvent(msg.Channel, []byte(msg.Payload))

			bmsg := &BroadcastMsg{
				Channel: ChannelName(msg.Channel),
				Payload: clientMsg,
			}

			select {
			case hub.broadcast <- bmsg:
			default:
				logger.S.Warnf("Hub broadcast channel full, dropping message for %s", msg.Channel)
			}
		}

		pubsub.Close()

		// If we reach here, the channel was closed (e.g., Redis disconnect)
		if hub.ctx.Err() == nil {
			logger.S.Warnf("Redis subscription for %s closed, reconnecting...", pattern)
			time.Sleep(1 * time.Second)
		}
	}
}

func formatClientEvent(redisChannel string, rawPayload []byte) (string, []byte) {
	var clientChannel string
	switch {
	case len(redisChannel) >= 7 && redisChannel[:7] == "player:":
		clientChannel = "player"
	case len(redisChannel) >= 6 && redisChannel[:6] == "trade:":
		clientChannel = "trade"
	case len(redisChannel) >= 6 && redisChannel[:6] == "notif:":
		clientChannel = "notification"
	case len(redisChannel) >= 7 && redisChannel[:7] == "combat:":
		clientChannel = "combat"
	case len(redisChannel) >= 10 && redisChannel[:10] == "chat:sect:":
		clientChannel = "chat:sect"
	default:
		clientChannel = redisChannel // "chat:world", "chat:general", "world:pos", "world:cell"
	}

	event := struct {
		Type    string          `json:"type"`
		Channel string          `json:"channel"`
		Data    json.RawMessage `json:"data"`
	}{
		Type:    "event",
		Channel: clientChannel,
		Data:    json.RawMessage(rawPayload),
	}

	out, err := json.Marshal(event)
	if err != nil {
		return clientChannel, rawPayload
	}
	return clientChannel, out
}
