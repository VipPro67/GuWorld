package handlers

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/cache"
)

type msgHistoryItem struct {
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

var (
	// In-memory fallback if Redis is nil/offline
	memChatHistory   = make(map[string][]msgHistoryItem)
	memChatHistoryMu sync.Mutex

	// Vietnamese profanity single words (normalized, lowercase)
	profanityList = map[string]bool{
		"dm": true, "đm": true, "dmm": true, "đmm": true, "dmmm": true, "đmmm": true,
		"cl": true, "cln": true, "vcl": true, "vkl": true, "vl": true, "vler": true, "vlxx": true, "vclxx": true,
		"cặc": true, "cac": true, "cack": true,
		"lồn": true, "lon": true, "lờ": true, "lozn": true, "l0n": true,
		"đéo": true, "deo": true, "đeo": true,
		"địt": true, "dit": true, "djt": true, "đjt": true, "đệt": true, "det": true,
		"đĩ": true, "di": true, "điếm": true, "diem": true,
		"buồi": true, "buoi": true, "bùi": true, "bui": true,
		"bướm": true, "buom": true,
		"cứt": true, "cut": true,
		"cmn": true,
	}

	// Vietnamese profanity phrases (2 or more words, lowercase, space-separated)
	profanityPhrases = map[string]bool{
		"óc chó": true, "oc cho": true,
		"chó đẻ": true, "cho de": true,
		"chó má": true, "cho ma": true,
		"súc vật": true, "suc vat": true,
		"mẹ kiếp": true, "me kiep": true,
		"hãm lồn": true, "ham lon": true,
		"ngu lồn": true, "ngu lon": true,
		"bú cu": true, "bu cu": true,
		"bú lồn": true, "bu lon": true,
		"đầu buồi": true, "dau buoi": true,
		"mẹ mày": true, "me may": true,
		"địt mẹ": true, "dit me": true,
		"dịt mẹ": true, "đm mày": true,
		"dm may": true, "đmm mày": true,
		"dmm may": true, "con mẹ": true,
		"con me": true, "đồ chó": true,
		"do cho": true, "ăn cứt": true,
		"an cut": true,
	}
)

// getChatHistory retrieves player chat history from Redis, falling back to memory
func getChatHistory(playerID string) []msgHistoryItem {
	var history []msgHistoryItem
	if cache.Client != nil {
		err := cache.GetJSON(fmt.Sprintf("chat_history:%s", playerID), &history)
		if err == nil {
			return history
		}
	}

	memChatHistoryMu.Lock()
	defer memChatHistoryMu.Unlock()
	if items, exists := memChatHistory[playerID]; exists {
		copied := make([]msgHistoryItem, len(items))
		copy(copied, items)
		return copied
	}
	return nil
}

// saveChatHistory saves player chat history to Redis and memory
func saveChatHistory(playerID string, history []msgHistoryItem) {
	if cache.Client != nil {
		_ = cache.SetJSON(fmt.Sprintf("chat_history:%s", playerID), history, 15*time.Second)
	}

	memChatHistoryMu.Lock()
	memChatHistory[playerID] = history
	memChatHistoryMu.Unlock()
}

// CleanAndNormalizeText removes punctuation, normalizes letters to lowercase and collapses spaces
func CleanAndNormalizeText(text string) string {
	text = strings.ToLower(text)
	// Replace standard punctuations with space to avoid merging words together
	replacer := strings.NewReplacer(
		".", " ", ",", " ", "!", " ", "?", " ", "-", " ", "_", " ", "*", " ", "~", " ",
		"/", " ", "\\", " ", "@", " ", "#", " ", "$", " ", "%", " ", "^", " ", "&", " ",
		"(", " ", ")", " ", "[", " ", "]", " ", "{", " ", "}", " ", "|", " ", "\n", " ",
		"\t", " ", "\"", " ", "'", " ", ":", " ", ";", " ",
	)
	text = replacer.Replace(text)
	return strings.Join(strings.Fields(text), " ")
}

// ValidateChatMessage checks if the chat content is spam or contains profanity
func ValidateChatMessage(playerID string, content string) error {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return fmt.Errorf("Tin nhắn không được để trống!")
	}

	now := time.Now()

	// Fetch sliding window chat history
	history := getChatHistory(playerID)

	// Filter history to keep only entries from the last 10 seconds
	var activeHistory []msgHistoryItem
	for _, item := range history {
		if now.Sub(item.Timestamp) < 10*time.Second {
			activeHistory = append(activeHistory, item)
		}
	}

	// 1. Check Rate Limit (based on the most recent message)
	if len(activeHistory) > 0 {
		lastMsg := activeHistory[len(activeHistory)-1]
		if now.Sub(lastMsg.Timestamp) < 1500*time.Millisecond {
			return fmt.Errorf("Thao tác quá nhanh! Vui lòng chờ một lát.")
		}
	}

	// 2. Check duplicate spam (consecutive & sliding window)
	if len(activeHistory) > 0 {
		// Rule 2a: Consecutive duplicate check (immediate previous message is identical)
		lastMsg := activeHistory[len(activeHistory)-1]
		if lastMsg.Content == trimmed {
			return fmt.Errorf("Không được gửi tin nhắn trùng lặp trong thời gian ngắn!")
		}

		// Rule 2b: Sliding window check (same message sent >= 2 times in the last 10s, this would be the 3rd)
		duplicateCount := 0
		for _, item := range activeHistory {
			if item.Content == trimmed {
				duplicateCount++
			}
		}
		if duplicateCount >= 2 {
			return fmt.Errorf("Bạn đang gửi cùng một tin nhắn quá nhiều lần trong thời gian ngắn!")
		}
	}

	// 3. Check for repeating character sequences (e.g. "aaaaaaa...")
	if hasRepeatingCharacters(trimmed, 10) {
		return fmt.Errorf("Tin nhắn chứa chuỗi ký tự lặp lại quá nhiều (spam)!")
	}

	// Normalize text for word check
	normalized := CleanAndNormalizeText(trimmed)
	words := strings.Fields(normalized)

	if len(words) == 0 {
		return nil
	}

	// 4. Check for consecutively repeated words (e.g. "alo alo alo alo")
	consecutiveLimit := 4
	consecutiveCount := 1
	var lastWord string
	for _, w := range words {
		if w == lastWord {
			consecutiveCount++
			if consecutiveCount >= consecutiveLimit {
				return fmt.Errorf("Phát hiện spam! Vui lòng không lặp lại từ liên tục.")
			}
		} else {
			consecutiveCount = 1
			lastWord = w
		}
	}

	// 5. Check for profanity
	// Check single words
	for _, w := range words {
		if profanityList[w] {
			return fmt.Errorf("Tin nhắn chứa từ ngữ nhạy cảm hoặc nói tục!")
		}
	}

	// Check 2-word phrases
	for i := 0; i < len(words)-1; i++ {
		phrase := words[i] + " " + words[i+1]
		if profanityPhrases[phrase] {
			return fmt.Errorf("Tin nhắn chứa từ ngữ nhạy cảm hoặc nói tục!")
		}
		// Also check stripped 2-word phrases just in case
		strippedPhrase := words[i] + words[i+1]
		if profanityList[strippedPhrase] {
			return fmt.Errorf("Tin nhắn chứa từ ngữ nhạy cảm hoặc nói tục!")
		}
	}

	// Append new message to active history and save
	activeHistory = append(activeHistory, msgHistoryItem{
		Content:   trimmed,
		Timestamp: now,
	})
	saveChatHistory(playerID, activeHistory)

	return nil
}

// hasRepeatingCharacters checks if there is any character repeated >= limit times consecutively
func hasRepeatingCharacters(text string, limit int) bool {
	if len(text) < limit {
		return false
	}
	runes := []rune(text)
	count := 1
	var prev rune
	for i, r := range runes {
		if i == 0 {
			prev = r
			continue
		}
		if r == prev {
			count++
			if count >= limit {
				return true
			}
		} else {
			count = 1
			prev = r
		}
	}
	return false
}
