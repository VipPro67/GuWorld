// Package uihub is the transport-neutral pub/sub bus for player UI events.
// It no longer owns any network connection: WebTransport UI streams register a
// MessageSink, and Redis subscribers fan messages into the same channels.
package uihub

import (
	"context"
	"hash/fnv"
	"sync"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// ChannelName provides stronger typing for channel keys.
type ChannelName string

// MessageSink receives broadcast payloads for a subscribed channel.
// Deliver must not block the broadcast loop.
type MessageSink interface {
	Deliver(payload []byte)
}

// SinkFunc adapts a function to MessageSink.
type SinkFunc func(payload []byte)

func (f SinkFunc) Deliver(payload []byte) {
	if f != nil {
		f(payload)
	}
}

// BroadcastMsg is the envelope used internally by the Hub to fan a payload out
// to subscribers of one channel.
type BroadcastMsg struct {
	Channel ChannelName
	Payload []byte
	// Exclude is the player ID to skip (e.g., the sender).
	Exclude uuid.UUID
	// Targets restricts broadcast to only these player IDs.
	// If nil/empty, broadcasts to all subscribers of the channel.
	Targets map[uuid.UUID]struct{}
}

// Hub owns the channel subscriptions for one shard. All delivery is async: a
// slow sink never blocks another subscriber because the broadcast loop snapshots
// the subscriber list and calls sinks outside the lock.
type Hub struct {
	mu        sync.RWMutex
	channels  map[ChannelName]map[uuid.UUID]MessageSink
	broadcast chan *BroadcastMsg

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// InitHub creates and starts a hub shard.
func InitHub() *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	h := &Hub{
		channels:  make(map[ChannelName]map[uuid.UUID]MessageSink),
		broadcast: make(chan *BroadcastMsg, 4096),
		ctx:       ctx,
		cancel:    cancel,
	}
	h.wg.Add(1)
	go h.run()
	return h
}

func (h *Hub) run() {
	defer func() {
		if r := recover(); r != nil {
			logger.S.Errorf("UI hub run panic recovered: %v", r)
		}
		h.wg.Done()
	}()
	for {
		select {
		case <-h.ctx.Done():
			h.mu.Lock()
			h.channels = make(map[ChannelName]map[uuid.UUID]MessageSink)
			h.mu.Unlock()
			return

		case msg := <-h.broadcast:
			h.mu.RLock()
			subs := h.channels[msg.Channel]
			if len(subs) == 0 {
				h.mu.RUnlock()
				continue
			}
			targets := make([]MessageSink, 0, len(subs))
			for pid, sink := range subs {
				if msg.Exclude != uuid.Nil && pid == msg.Exclude {
					continue
				}
				if msg.Targets != nil {
					if _, included := msg.Targets[pid]; !included {
						continue
					}
				}
				targets = append(targets, sink)
			}
			h.mu.RUnlock()
			for _, sink := range targets {
				sink.Deliver(msg.Payload)
			}
		}
	}
}

// Stop gracefully shuts down the hub.
func (h *Hub) Stop(ctx context.Context) error {
	h.cancel()

	done := make(chan struct{})
	go func() {
		h.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// SubscribeSink registers a MessageSink for a player on a channel.
func (h *Hub) SubscribeSink(playerID uuid.UUID, channel ChannelName, sink MessageSink) {
	if h == nil || sink == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.channels[channel] == nil {
		h.channels[channel] = make(map[uuid.UUID]MessageSink)
	}
	h.channels[channel][playerID] = sink
}

// UnsubscribeSink removes a player's MessageSink from a channel.
func (h *Hub) UnsubscribeSink(playerID uuid.UUID, channel ChannelName) {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if subs, ok := h.channels[channel]; ok {
		delete(subs, playerID)
		if len(subs) == 0 {
			delete(h.channels, channel)
		}
	}
}

// UnsubscribeSinkIf removes a player's MessageSink from a channel only if it matches sink.
func (h *Hub) UnsubscribeSinkIf(playerID uuid.UUID, channel ChannelName, sink MessageSink) {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if subs, ok := h.channels[channel]; ok {
		if sink == nil || subs[playerID] == sink {
			delete(subs, playerID)
			if len(subs) == 0 {
				delete(h.channels, channel)
			}
		}
	}
}

// UnsubscribeAllSinks removes all sink subscriptions for a player on this hub.
func (h *Hub) UnsubscribeAllSinks(playerID uuid.UUID) {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch, subs := range h.channels {
		delete(subs, playerID)
		if len(subs) == 0 {
			delete(h.channels, ch)
		}
	}
}

// UnsubscribeAllSinksIf removes all sink subscriptions matching sink for a player on this hub.
func (h *Hub) UnsubscribeAllSinksIf(playerID uuid.UUID, sink MessageSink) {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch, subs := range h.channels {
		if sink == nil || subs[playerID] == sink {
			delete(subs, playerID)
			if len(subs) == 0 {
				delete(h.channels, ch)
			}
		}
	}
}

func (h *Hub) hasPlayer(playerID uuid.UUID) bool {
	if h == nil {
		return false
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, subs := range h.channels {
		if _, ok := subs[playerID]; ok {
			return true
		}
	}
	return false
}

// BroadcastToChannel sends a message to all subscribers of a channel, excluding the given player.
func (h *Hub) BroadcastToChannel(channel ChannelName, payload []byte, exclude uuid.UUID) {
	if h == nil {
		return
	}
	select {
	case h.broadcast <- &BroadcastMsg{Channel: channel, Payload: payload, Exclude: exclude}:
	default:
	}
}

// BroadcastToTargets sends a message to specific players in a channel.
func (h *Hub) BroadcastToTargets(channel ChannelName, payload []byte, targets map[uuid.UUID]struct{}) {
	if h == nil || targets != nil && len(targets) == 0 {
		return
	}
	select {
	case h.broadcast <- &BroadcastMsg{Channel: channel, Payload: payload, Targets: targets}:
	default:
	}
}

// GetStats returns the number of distinct subscribed players and active channels.
func (h *Hub) GetStats() (int, int) {
	if h == nil {
		return 0, 0
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	channels := len(h.channels)
	players := make(map[uuid.UUID]struct{})
	for _, subs := range h.channels {
		for pid := range subs {
			players[pid] = struct{}{}
		}
	}
	return len(players), channels
}

// IsConnected reports whether a player currently has at least one UI subscription.
func (h *Hub) IsConnected(playerID uuid.UUID) bool {
	return h.hasPlayer(playerID)
}

// ShardCount defines the number of hub shards for parallel processing.
const ShardCount = 16
const shardMask = ShardCount - 1

// ShardedHub distributes channel subscriptions across independent Hub shards.
type ShardedHub struct {
	shards [ShardCount]*Hub
}

var GlobalShardedHub *ShardedHub

// InitShardedHub creates and starts the global sharded hub.
func InitShardedHub() *ShardedHub {
	sh := &ShardedHub{}
	for i := 0; i < ShardCount; i++ {
		sh.shards[i] = InitHub()
	}
	GlobalShardedHub = sh
	logger.S.Infof("Realtime UI hub initialized with %d shards", ShardCount)
	return sh
}

func shardForChannel(channel ChannelName) int {
	h := fnv.New32()
	h.Write([]byte(channel))
	return int(h.Sum32()) & shardMask
}

// GetChannelShard returns the shard that handles a channel's subscriptions.
func (sh *ShardedHub) GetChannelShard(channel ChannelName) *Hub {
	return sh.shards[shardForChannel(channel)]
}

// SubscribeSink registers a MessageSink for a player on the channel's shard.
func (sh *ShardedHub) SubscribeSink(playerID uuid.UUID, channel ChannelName, sink MessageSink) {
	if sh == nil {
		return
	}
	sh.GetChannelShard(channel).SubscribeSink(playerID, channel, sink)
}

// UnsubscribeSink removes a player's MessageSink from the channel's shard.
func (sh *ShardedHub) UnsubscribeSink(playerID uuid.UUID, channel ChannelName) {
	if sh == nil {
		return
	}
	sh.GetChannelShard(channel).UnsubscribeSink(playerID, channel)
}

// UnsubscribeSinkIf removes a player's MessageSink from the channel's shard only if it matches sink.
func (sh *ShardedHub) UnsubscribeSinkIf(playerID uuid.UUID, channel ChannelName, sink MessageSink) {
	if sh == nil {
		return
	}
	sh.GetChannelShard(channel).UnsubscribeSinkIf(playerID, channel, sink)
}

// UnsubscribeAllSinks removes all sink subscriptions for a player across all shards.
func (sh *ShardedHub) UnsubscribeAllSinks(playerID uuid.UUID) {
	if sh == nil {
		return
	}
	for i := 0; i < ShardCount; i++ {
		sh.shards[i].UnsubscribeAllSinks(playerID)
	}
}

// UnsubscribeAllSinksIf removes all sink subscriptions matching sink for a player across all shards.
func (sh *ShardedHub) UnsubscribeAllSinksIf(playerID uuid.UUID, sink MessageSink) {
	if sh == nil {
		return
	}
	for i := 0; i < ShardCount; i++ {
		sh.shards[i].UnsubscribeAllSinksIf(playerID, sink)
	}
}

// BroadcastToChannel sends a message to all subscribers of a channel.
func (sh *ShardedHub) BroadcastToChannel(channel ChannelName, payload []byte, exclude uuid.UUID) {
	if sh == nil {
		return
	}
	sh.GetChannelShard(channel).BroadcastToChannel(channel, payload, exclude)
}

// BroadcastToTargets sends a message to specific players in a channel.
func (sh *ShardedHub) BroadcastToTargets(channel ChannelName, payload []byte, targets map[uuid.UUID]struct{}) {
	if sh == nil || targets != nil && len(targets) == 0 {
		return
	}
	sh.GetChannelShard(channel).BroadcastToTargets(channel, payload, targets)
}

// IsConnected reports whether a player currently has at least one UI subscription.
func (sh *ShardedHub) IsConnected(playerID uuid.UUID) bool {
	if sh == nil {
		return false
	}
	for i := 0; i < ShardCount; i++ {
		if sh.shards[i].hasPlayer(playerID) {
			return true
		}
	}
	return false
}

// GetStats returns total subscribed players and active channels across all shards.
func (sh *ShardedHub) GetStats() (int, int) {
	if sh == nil {
		return 0, 0
	}
	var totalClients, totalChannels int
	for i := 0; i < ShardCount; i++ {
		clients, channels := sh.shards[i].GetStats()
		totalClients += clients
		totalChannels += channels
	}
	return totalClients, totalChannels
}

// Stop gracefully shuts down all shards.
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
