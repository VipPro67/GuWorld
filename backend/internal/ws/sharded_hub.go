package ws

import (
	"context"
	"hash/fnv"
	"sync"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// ShardCount defines the number of hub shards for parallel processing.
// Power of 2 for fast modulo via bitwise AND.
const ShardCount = 16
const shardMask = ShardCount - 1

// ShardedHub distributes WebSocket clients and broadcasts across multiple
// independent Hub instances to eliminate the single-goroutine bottleneck.
type ShardedHub struct {
	shards [ShardCount]*Hub
}

var GlobalShardedHub *ShardedHub

// InitShardedHub creates and starts the global sharded WebSocket hub.
func InitShardedHub() *ShardedHub {
	sh := &ShardedHub{}
	for i := 0; i < ShardCount; i++ {
		sh.shards[i] = InitHub()
	}
	GlobalShardedHub = sh
	logger.S.Infof("Sharded WebSocket Hub initialized with %d shards", ShardCount)
	return sh
}

// shardFor returns the shard index for a given UUID using consistent hashing.
func shardFor(id uuid.UUID) int {
	h := fnv.New32()
	h.Write(id[:])
	return int(h.Sum32()) & shardMask
}

// shardForChannel returns the shard index for a channel name.
// Player-specific channels route to the shard based on channel hash.
func shardForChannel(channel ChannelName) int {
	h := fnv.New32()
	h.Write([]byte(channel))
	return int(h.Sum32()) & shardMask
}

// GetClientShard returns the shard that holds a client's registration.
func (sh *ShardedHub) GetClientShard(playerID uuid.UUID) *Hub {
	return sh.shards[shardFor(playerID)]
}

// GetChannelShard returns the shard that handles a channel's subscriptions.
func (sh *ShardedHub) GetChannelShard(channel ChannelName) *Hub {
	return sh.shards[shardForChannel(channel)]
}

// Register adds a client to the appropriate shard.
func (sh *ShardedHub) Register(client *Client) {
	shard := sh.GetClientShard(client.playerID)
	shard.register <- client
}

// Unregister removes a client from the appropriate shard.
func (sh *ShardedHub) Unregister(client *Client) {
	shard := sh.GetClientShard(client.playerID)
	shard.unregister <- client
}

// Subscribe adds a client to a channel on the appropriate shard.
func (sh *ShardedHub) Subscribe(client *Client, channel ChannelName) {
	shard := sh.GetChannelShard(channel)
	shard.Subscribe(client, channel)
}

// BroadcastToChannel sends a message to all subscribers of a channel.
func (sh *ShardedHub) BroadcastToChannel(channel ChannelName, payload []byte, exclude uuid.UUID) {
	shard := sh.GetChannelShard(channel)
	shard.BroadcastToChannel(channel, payload, exclude)
}

// BroadcastToTargets sends a message to specific players in a channel.
func (sh *ShardedHub) BroadcastToTargets(channel ChannelName, payload []byte, targets map[uuid.UUID]struct{}) {
	shard := sh.GetChannelShard(channel)
	shard.BroadcastToTargets(channel, payload, targets)
}

// IsConnected reports whether a player currently has an active WebSocket connection.
func (sh *ShardedHub) IsConnected(playerID uuid.UUID) bool {
	if sh == nil {
		return false
	}
	shard := sh.GetClientShard(playerID)
	return shard.IsConnected(playerID)
}

// GetStats returns total connected clients and active channels across all shards.
func (sh *ShardedHub) GetStats() (int, int) {
	if sh == nil {
		return 0, 0
	}
	var totalClients, totalChannels int
	var wg sync.WaitGroup
	wg.Add(ShardCount)
	for i := 0; i < ShardCount; i++ {
		go func(idx int) {
			defer wg.Done()
			clients, channels := sh.shards[idx].GetStats()
			totalClients += clients
			totalChannels += channels
		}(i)
	}
	wg.Wait()
	return totalClients, totalChannels
}

// Stop gracefully shuts down all shards and disconnects clients.
func (sh *ShardedHub) Stop(ctx context.Context) error {
	var wg sync.WaitGroup
	wg.Add(ShardCount)
	for i := 0; i < ShardCount; i++ {
		go func(idx int) {
			defer wg.Done()
			_ = sh.shards[idx].Stop(ctx)
		}(i)
	}
	wg.Wait()
	return nil
}
