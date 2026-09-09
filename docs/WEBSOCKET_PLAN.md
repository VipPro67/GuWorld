# WebSocket Implementation Plan

## Overview
Replace SSE (Server-Sent Events) with WebSocket for real-time communication. Use Redis Pub/Sub for horizontal scaling and reliable message delivery.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         FRONTEND                                │
│                                                                 │
│  ┌──────────────┐      ┌──────────────────────────────────┐    │
│  │ useGameStore │─────▶│     WebSocket Manager             │    │
│  │ (Zustand)    │      │  • connect(token)                 │    │
│  │              │      │  • subscribe(channel, callback)   │    │
│  │              │      │  • send(channel, payload)         │    │
│  │              │      │  • auto-reconnect (exp backoff)   │    │
│  └──────────────┘      └──────────────┬───────────────────┘    │
│                                        │                        │
└────────────────────────────────────────┼────────────────────────┘
                                         │ ws://host/ws
┌────────────────────────────────────────┼────────────────────────┐
│                        BACKEND         │                        │
│                                        ▼                        │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │               WebSocket Hub (gorilla/websocket)          │   │
│  │                                                          │   │
│  │  clients: map[playerID]*Client                           │   │
│  │  channels: map[channelName]map[playerID]*Client          │   │
│  │                                                          │   │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐       │   │
│  │  │ player  │ │  chat   │ │  trade  │ │  world  │       │   │
│  │  │ channel │ │ channels│ │ channel │ │ channel │       │   │
│  │  └─────────┘ └─────────┘ └─────────┘ └─────────┘       │   │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐                   │   │
│  │  │ combat  │ │  notif  │ │  mail   │                   │   │
│  │  │ channel │ │ channel │ │ channel │                   │   │
│  │  └─────────┘ └─────────┘ └─────────┘                   │   │
│  └──────────────────────────┬──────────────────────────────┘   │
│                             │                                   │
│              ┌──────────────▼──────────────┐                   │
│              │     Redis Pub/Sub            │                   │
│              │  • player:{id}               │                   │
│              │  • chat:{channel}            │                   │
│              │  • trade:{id}                │                   │
│              │  • world:pos                 │                   │
│              │  • world:cell                │                   │
│              │  • combat:{id}               │                   │
│              │  • notif:{id}                │                   │
│              └──────────────┬──────────────┘                   │
│                             │                                   │
│              ┌──────────────▼──────────────┐                   │
│              │   Existing Handlers           │                   │
│              │   (fat handlers不变, chỉ      │                   │
│              │    thay SSE→Redis Publish)    │                   │
│              └─────────────────────────────┘                   │
└─────────────────────────────────────────────────────────────────┘
```

## Phase 1: Backend — WebSocket Hub Core

### Files to Create

| File | Purpose | Est. Lines |
|------|---------|------------|
| `backend/internal/ws/hub.go` | Connection manager | ~200 |
| `backend/internal/ws/client.go` | Per-connection pumps | ~150 |
| `backend/internal/ws/message.go` | Message types | ~80 |
| `backend/internal/ws/redis_sub.go` | Redis subscriber goroutines | ~120 |
| `backend/internal/ws/auth.go` | JWT auth for WS upgrade | ~60 |

### Hub Structure

```go
// hub.go
type Hub struct {
    mu       sync.RWMutex
    clients  map[uuid.UUID]*Client
    channels map[string]map[uuid.UUID]*Client
    
    register   chan *Client
    unregister chan *Client
    broadcast  chan *BroadcastMsg
}

type Client struct {
    playerID uuid.UUID
    conn     *websocket.Conn
    send     chan []byte
    hub      *Hub
    subscriptions map[string]struct{}
}

type BroadcastMsg struct {
    Channel string
    Payload []byte
    Exclude uuid.UUID
}
```

### Message Protocol (message.go)

```go
type WSMessage struct {
    Type    string          `json:"type"`
    Channel string          `json:"channel,omitempty"`
    Payload json.RawMessage `json:"payload,omitempty"`
}

// Client → Server
type SubscribeMsg struct { Channel string `json:"channel"` }
type ChatSendMsg struct { Content string `json:"content"` }

// Server → Client
type EventMsg struct {
    Type    string      `json:"type"`
    Channel string      `json:"channel"`
    Data    interface{} `json:"data"`
}
```

### Channels

| Channel | Direction | Purpose |
|---------|-----------|---------|
| `player` | S→C | Player state updates |
| `chat:world` | Both | World chat |
| `chat:general` | Both | Bảo Hoàng Thiên chat |
| `chat:sect` | Both | Sect chat |
| `trade` | S→C | Trade state updates |
| `world:pos` | S→C | Player position |
| `world:cell` | S→C | Cell clear/respawn |
| `combat` | S→C | Combat events |
| `notification` | S→C | Push notifications |
| `mail` | S→C | Private mail |

## Phase 2: Backend — Redis Pub/Sub Integration

### Redis Channel Naming

| Redis Channel | Pattern | Description |
|---------------|---------|-------------|
| `player:{playerID}` | Per-player | Player state updates |
| `chat:{channel}` | Per-channel | Chat messages |
| `trade:{playerID}` | Per-player | Trade state |
| `world:pos` | Global | Player positions |
| `world:cell` | Global | Cell events |
| `combat:{playerID}` | Per-player | Combat state |
| `notif:{playerID}` | Per-player | Notifications |

### cache/redis.go additions

```go
func PublishEvent(channel string, payload interface{}) error
func SubscribeChannels(ctx context.Context, channels ...string) *redis.PubSub
```

### Redis Subscriber (redis_sub.go)

```go
func StartRedisSubscribers(hub *Hub) {
    go subscribePattern(hub, "player:*")
    go subscribePattern(hub, "chat:*")
    go subscribePattern(hub, "trade:*")
    go subscribePattern(hub, "world:pos")
    go subscribePattern(hub, "world:cell")
    go subscribePattern(hub, "combat:*")
    go subscribePattern(hub, "notif:*")
}
```

## Phase 3: Backend — Handler Integration

### Pattern Change

```go
// BEFORE (SSE)
func notifyBaoHoangMessages(channel string, sectID sql.NullString) {
    messages := loadBaoHoangMessages(channel, sectID)
    chatEventHub.Lock()
    for ch := range chatEventHub.subs[key] {
        select { case ch <- payload: default: }
    }
    chatEventHub.Unlock()
}

// AFTER (Redis Pub/Sub)
func notifyBaoHoangMessages(channel string, sectID sql.NullString) {
    messages := loadBaoHoangMessages(channel, sectID)
    key := chatChannelKey(channel, sectID)
    cache.PublishEvent("chat:"+key, chatEventPayload{Messages: messages})
}
```

### Files to Modify

| File | Changes |
|------|---------|
| `handlers/player_events.go` | Remove SSE endpoints, publish to Redis |
| `handlers/chat_events.go` | Remove in-memory hub, publish to Redis |
| `handlers/trade_events.go` | Remove in-memory hub, publish to Redis |
| `handlers/world_events.go` | Remove in-memory hub, publish to Redis |
| `handlers/world_cell_events.go` | Remove in-memory hub, publish to Redis |
| `pkg/notify/notify.go` | Remove in-memory hub, publish to Redis |
| `server/server.go` | Add `/ws` route, remove 4 SSE routes |

### Files to Remove Code From

| Code | File |
|------|------|
| `chatEventHub`, `subscribeChat`, `unsubscribeChat` | `chat_events.go` |
| `tradeEventHub`, `subscribeTrade`, `unsubscribeTrade` | `trade_events.go` |
| `worldPosHub`, `subscribeWorldPos`, `unsubscribeWorldPos` | `world_events.go` |
| `worldCellHub`, `subscribeWorldCell`, `unsubscribeWorldCell` | `world_cell_events.go` |
| `notify.hub`, `notify.Subscribe`, `notify.Unsubscribe` | `notify.go` |
| `StreamUnifiedEvents`, `StreamPlayerEvents` | `player_events.go` |
| `SubscribeTradeEvents` | `trade_events.go` |
| `SubscribeBaoHoangMessages` | `chat_events.go` |

## Phase 4: Backend — Cleanup & Config

### go.mod
```bash
go get github.com/gorilla/websocket@latest
```

### server.go — Compression Skip List
Remove SSE routes from compression exclude:
```go
// REMOVE these cases
case "/api/events", "/api/player/events",
    "/api/trade/events", "/api/chat/bao-hoang/events":
    return true
```

## Phase 5: Frontend — WebSocket Manager

### File: `frontend/src/lib/websocket.ts`

```typescript
class WebSocketManager {
  private ws: WebSocket | null = null;
  private url: string;
  private token: string;
  private reconnectAttempts = 0;
  private maxReconnectDelay = 30000;
  private subscriptions = new Map<string, Set<(data: any) => void>>();
  
  constructor(url: string) { this.url = url; }
  
  connect(token: string): Promise<void>
  subscribe(channel: string, callback: (data: any) => void): () => void
  send(channel: string, payload: any)
  private reconnect()
}

export const wsManager = new WebSocketManager('ws://localhost:8080/ws');
```

## Phase 6: Frontend — Store Integration

### useGameStore.ts Changes

```typescript
// BEFORE — SSE
subscribeEvents: () => {
  const source = new EventSource(`${BACKEND_URL}/api/events?token=${token}`);
  source.addEventListener('chat.messages.world', ...);
  source.addEventListener('player.updated', ...);
  // 10+ listeners
}

// AFTER — WebSocket
subscribeEvents: async () => {
  await wsManager.connect(token);
  
  wsManager.subscribe('player', (data) => {
    set((state) => ({ player: { ...state.player, ...data } }));
  });
  
  wsManager.subscribe('chat:world', (messages) => {
    set((state) => ({
      chatMessages: { ...state.chatMessages, world: messages },
      unreadChannels: { ...state.unreadChannels, world: state.activeChannel !== 'world' },
      baoHoangMessages: state.activeChannel === 'world' ? messages : state.baoHoangMessages,
    }));
  });
  // ... same for chat:general, chat:sect
  
  wsManager.subscribe('trade', (data) => set({ activeTrade: data.trade }));
  wsManager.subscribe('notification', (n) => set({
    notifications: [n, ...state.notifications].slice(0, 50),
    unreadNotifications: state.unreadNotifications + 1,
  }));
  
  wsManager.subscribe('world:pos', (data) => 
    window.dispatchEvent(new CustomEvent('gu_world_pos', { detail: data }))
  );
  wsManager.subscribe('world:cell', (data) => 
    window.dispatchEvent(new CustomEvent('gu_world_cell', { detail: data }))
  );
}
```

### Chat Send — Still REST
```typescript
// Keep REST for validation + rate limit
sendBaoHoangChat: async (content, channel) => {
  const response = await authFetch(`${BACKEND_URL}/api/chat/bao-hoang/send`, {
    method: "POST",
    body: JSON.stringify({ content, channel }),
  });
  // Server publishes to Redis → WS broadcast
}
```

### Layout — Connection Lifecycle
```typescript
useEffect(() => {
  if (token) {
    wsManager.connect(token);
    subscribeEvents();
  }
  return () => wsManager.disconnect();
}, [token]);
```

## Phase 7: Testing

```bash
# Backend
cd backend && go test ./...
cd backend && go test -race ./...
cd backend && go build ./cmd/server

# Frontend
cd frontend && npm run lint
cd frontend && npm run build
```

### Manual Checklist
- [ ] Connect WS, receive player snapshot
- [ ] Send chat → all subscribers receive
- [ ] Move player → nearby see position
- [ ] Trade invite → both see update
- [ ] Kill boss → world cell event
- [ ] Notification → bell icon updates
- [ ] Disconnect → auto-reconnect
- [ ] Multiple tabs → 1 connection shared

## Phase 8: Rollback Strategy

Keep 4 SSE endpoints as deprecated fallback:
```go
func StreamUnifiedEvents(c *gin.Context) {
    // DEPRECATED: Use WebSocket /ws instead
    // Keep old logic for fallback
}
```

## Timeline

| Phase | Effort | Priority |
|-------|--------|----------|
| 1-2: Backend WS Hub + Redis | 3-4 days | Core |
| 3-4: Handler Integration + Cleanup | 2-3 days | Core |
| 5: go.mod | 5 min | Quick |
| 6: Frontend | 3-4 days | Core |
| 7-8: Testing + Rollback | 2 days | Safety |
| **Total** | **~10-12 days** | |