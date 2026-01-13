package ws

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
)

// Hub maintains the set of active clients and broadcasts messages.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound audio chunks from the speaker.
	AudioChan chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	// Lock for safe map access
	mu sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		AudioChan:  make(chan []byte, 100), // Buffered channel for audio
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("Client connected: %s", client.conn.RemoteAddr())
			h.BroadcastStats()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("Client disconnected")
			h.BroadcastStats()
		}
	}
}

// StatsMessage defines the structure of the stats update
type StatsMessage struct {
	Type      string         `json:"type"`
	Total     int            `json:"total"`
	Languages map[string]int `json:"languages"`
}

// BroadcastStats calculates and sends connection statistics to all clients
func (h *Hub) BroadcastStats() {
	h.mu.Lock()
	total := 0
	langs := make(map[string]int)

	for client := range h.clients {
		// Only count listeners? For now, count everyone, or filter if we add client types.
		// Assuming speakers also have a language set or empty.
		// Let's count everyone for simplicity, or we can filter if client has a specific flag.
		// Since the requirements implies "listeners", but currently we don't strictly distinguish roles in backend
		// other than by behavior (sending audio vs receiving text).

		total++
		l := client.TargetLang
		if l == "" {
			l = "connected" // Default or Speaker
		}
		langs[l]++
	}
	h.mu.Unlock()

	stats := StatsMessage{
		Type:      "stats",
		Total:     total,
		Languages: langs,
	}

	payload, err := json.Marshal(stats)
	if err != nil {
		log.Printf("Error marshaling stats: %v", err)
		return
	}

	h.BroadcastText(payload)
}

// BroadcastText sends a message to all connected clients
func (h *Hub) BroadcastText(message []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for client := range h.clients {
		select {
		case client.send <- message:
		default:
			close(client.send)
			delete(h.clients, client)
		}
	}
}

// BroadcastWithTranslation sends translated text based on client preference
func (h *Hub) BroadcastWithTranslation(originalText string, translateFunc func(text, target string) string) {
	// 1. Snapshot clients and languages to release lock quickly
	//    or just process everything under lock? Processing under lock with network calls is bad.
	//    Let's copy the clients.
	h.mu.Lock()
	type clientInfo struct {
		c    *Client
		lang string
	}
	var activeClients []clientInfo
	for c := range h.clients {
		activeClients = append(activeClients, clientInfo{c: c, lang: c.TargetLang})
	}
	h.mu.Unlock()

	// 2. Determine unique languages needed
	// Map lang -> translatedText
	translations := make(map[string]string)
	// Optimize: group unique langs
	uniqueLangs := make(map[string]bool)
	for _, ci := range activeClients {
		lang := ci.lang
		if lang == "" {
			lang = "zh" // Default source
		}
		uniqueLangs[lang] = true
	}

	// 3. Translate for each language
	for lang := range uniqueLangs {
		if lang == "zh" {
			translations[lang] = originalText
		} else {
			translations[lang] = translateFunc(originalText, lang)
		}
	}

	// 4. Send
	// We need to lock again? Or just send?
	// The client might have been disconnected and removed from 'clients' map in the meantime.
	// But 'close(client.send)' happens in unregister.
	// If we send to a closed channel, it panics.
	// We need to be careful.
	// Safer to lock again and check existence?

	h.mu.Lock()
	defer h.mu.Unlock()
	for _, ci := range activeClients {
		// Check if still connected
		if _, ok := h.clients[ci.c]; !ok {
			continue
		}

		lang := ci.lang
		if lang == "" {
			lang = "zh"
		}

		msg := translations[lang]
		// Construct HTMX update or JSON?
		// Let's send a simple JSON for the Listener to parse, or HTML snippet.
		// For robustness, let's send JSON: {"text": "...", "lang": "..."}

		// Wait, if front-end expects HTMX, we should send HTML.
		// "<div id='transcript-container' hx-swap-oob='beforeend'><p>Text</p></div>"
		// The prompt says "frontend uses HTML+CSS+HTMX+websocket".
		// We can append to the container using OOB swaps.

		payload := fmt.Sprintf(`<div hx-swap-oob="beforeend:#transcript-container"><div class="mb-2 p-2 bg-gray-800 rounded text-gray-200 border-l-4 border-indigo-500">%s</div></div>`, msg)

		select {
		case ci.c.send <- []byte(payload):
		default:
			close(ci.c.send)
			delete(h.clients, ci.c)
		}
	}
}
