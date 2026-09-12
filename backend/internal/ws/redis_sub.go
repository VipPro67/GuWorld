package ws

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// StartRedisSubscribers spawns goroutines to listen to Redis Pub/Sub channels
// and forward messages to the Sharded WebSocket Hub.
func StartRedisSubscribers(shub *ShardedHub) {
	go subscribePattern(shub, "player:*")
	go subscribePattern(shub, "chat:*")
	go subscribePattern(shub, "trade:*")
	go subscribePattern(shub, "world:pos")
	go subscribePattern(shub, "world:cell")
	go subscribePattern(shub, "world:cells:*")
	go subscribePattern(shub, "combat:*")
	go subscribePattern(shub, "notif:*")
	go subscribePattern(shub, "death_arena:*")
}

func subscribePattern(shub *ShardedHub, pattern string) {
	var backoffDelay time.Duration = 1 * time.Second
	const maxBackoffDelay = 30 * time.Second

	for {
		if shub == nil {
			return // Hub is stopping
		}

		if cache.Client == nil {
			time.Sleep(5 * time.Second)
			backoffDelay = 1 * time.Second // Reset backoff
			continue
		}

		pubsub := cache.Client.PSubscribe(context.Background(), pattern)
		// Wait for subscription to be created before moving on.
		_, err := pubsub.Receive(context.Background())
		if err != nil {
			logger.S.Warnf("Failed to subscribe to pattern %s: %v", pattern, err)
			pubsub.Close()

			// Exponential backoff with jitter
			jitter := time.Duration(float64(backoffDelay) * (0.5 + 0.5*float64(time.Now().UnixNano()%1000)/1000.0))
			time.Sleep(jitter)
			backoffDelay = min(backoffDelay*2, maxBackoffDelay)
			continue
		}

		logger.S.Infof("Subscribed to Redis pattern: %s", pattern)
		backoffDelay = 1 * time.Second // Reset backoff on successful subscribe

		ch := pubsub.Channel()

		// Receive loop
		for msg := range ch {
			clientChannel, clientMsg := formatClientEvent(msg.Channel, []byte(msg.Payload))

			if len(msg.Channel) >= 6 && msg.Channel[:6] == "trade:" {
				shub.BroadcastToChannel(ChannelName(msg.Channel), clientMsg, uuid.Nil)
				continue
			}

			shub.BroadcastToChannel(ChannelName(msg.Channel), clientMsg, uuid.Nil)
			if clientChannel != msg.Channel {
				shub.BroadcastToChannel(ChannelName(clientChannel), clientMsg, uuid.Nil)
			}
		}

		pubsub.Close()

		// If we reach here, the channel was closed (e.g., Redis disconnect)
		if shub != nil {
			logger.S.Warnf("Redis subscription for %s closed, reconnecting...", pattern)
			// Exponential backoff with jitter
			jitter := time.Duration(float64(backoffDelay) * (0.5 + 0.5*float64(time.Now().UnixNano()%1000)/1000.0))
			time.Sleep(jitter)
			backoffDelay = min(backoffDelay*2, maxBackoffDelay)
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
	case len(redisChannel) >= 12 && redisChannel[:12] == "death_arena:":
		clientChannel = "death_arena"
	case len(redisChannel) >= 10 && redisChannel[:10] == "chat:sect:":
		clientChannel = "chat:sect"
	case len(redisChannel) >= 11 && redisChannel[:11] == "world:cells:":
		clientChannel = "world:cells"
	default:
		clientChannel = redisChannel // "chat:world", "chat:general", "world:pos", "world:cell"
	}

	// Avoid double JSON marshaling: rawPayload is already JSON from Redis
	// Construct the event envelope without re-marshaling the payload
	event := struct {
		Type    string          `json:"type"`
		Channel string          `json:"channel"`
		Data    json.RawMessage `json:"data"`
	}{
		Type:    "event",
		Channel: clientChannel,
		Data:    rawPayload,
	}

	out, err := json.Marshal(event)
	if err != nil {
		return clientChannel, rawPayload
	}
	return clientChannel, out
}
