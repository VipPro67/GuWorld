package ws

import "encoding/json"

// WSMessage is the envelope for all WebSocket messages.
type WSMessage struct {
	Type      string          `json:"type"`
	Channel   string          `json:"channel,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Version   int             `json:"version,omitempty"`
	RequestID string          `json:"request_id,omitempty"`
}

// SubscribeMsg is sent by the client to subscribe to a channel.
type SubscribeMsg struct {
	Channel string `json:"channel"`
}

// ChatSendMsg is a generic chat message sent by the client.
type ChatSendMsg struct {
	Content string `json:"content"`
}

// EventMsg is the payload structure sent from Server to Client.
type EventMsg struct {
	Type    string      `json:"type"`
	Channel string      `json:"channel"`
	Data    interface{} `json:"data"`
}
