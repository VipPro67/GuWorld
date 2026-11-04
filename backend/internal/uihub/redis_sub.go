package uihub

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// StartRedisSubscribers spawns goroutines to listen to Redis Pub/Sub channels
// and forward messages to the sharded UI hub.
func StartRedisSubscribers(shub *ShardedHub) {
	go subscribePattern(shub, "player:*")
	go subscribePattern(shub, "refine:*")
	go subscribePattern(shub, "chat:*")
	go subscribePattern(shub, "trade:*")
	go subscribePattern(shub, "combat:*")
	go subscribePattern(shub, "notif:*")
	go subscribePattern(shub, "death_arena:*")
	go subscribePattern(shub, "martial_arena:*")
	go subscribePattern(shub, "catalog:*")
	go subscribePattern(shub, "world:*")
}

func subscribePattern(shub *ShardedHub, pattern string) {
	var backoffDelay time.Duration = 1 * time.Second
	const maxBackoffDelay = 30 * time.Second

	for {
		if shub == nil {
			return
		}

		if cache.Client == nil {
			time.Sleep(5 * time.Second)
			backoffDelay = 1 * time.Second
			continue
		}

		pubsub := cache.Client.PSubscribe(context.Background(), pattern)
		_, err := pubsub.Receive(context.Background())
		if err != nil {
			logger.S.Warnf("Failed to subscribe to pattern %s: %v", pattern, err)
			pubsub.Close()

			jitter := time.Duration(float64(backoffDelay) * (0.5 + 0.5*float64(time.Now().UnixNano()%1000)/1000.0))
			time.Sleep(jitter)
			backoffDelay = min(backoffDelay*2, maxBackoffDelay)
			continue
		}

		logger.S.Infof("Subscribed to Redis pattern: %s", pattern)
		backoffDelay = 1 * time.Second

		ch := pubsub.Channel()
		for msg := range ch {
			clientChannel, clientMsg := formatClientEvent(msg.Channel, []byte(msg.Payload))
			shub.BroadcastToChannel(ChannelName(msg.Channel), clientMsg, uuid.Nil)
			if clientChannel != msg.Channel {
				shub.BroadcastToChannel(ChannelName(clientChannel), clientMsg, uuid.Nil)
			}
		}

		pubsub.Close()

		if shub != nil {
			logger.S.Warnf("Redis subscription for %s closed, reconnecting...", pattern)
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
	case len(redisChannel) >= 7 && redisChannel[:7] == "refine:":
		clientChannel = "refine"
	case len(redisChannel) >= 6 && redisChannel[:6] == "trade:":
		clientChannel = "trade"
	case len(redisChannel) >= 6 && redisChannel[:6] == "notif:":
		clientChannel = "notification"
	case len(redisChannel) >= 7 && redisChannel[:7] == "combat:":
		clientChannel = "combat"
	case len(redisChannel) >= 12 && redisChannel[:12] == "death_arena:":
		clientChannel = "death_arena"
	case len(redisChannel) >= 14 && redisChannel[:14] == "martial_arena:":
		clientChannel = "martial_arena"
	case len(redisChannel) >= 8 && redisChannel[:8] == "catalog:":
		clientChannel = "catalog"
	default:
		clientChannel = redisChannel
	}

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
