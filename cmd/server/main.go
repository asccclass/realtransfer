package main

import (
	"log"
	"net/http"
	"path/filepath"
	"time"

	"realtransfer/internal/audio"
	"realtransfer/internal/docker"
	"realtransfer/internal/translate"
	"realtransfer/internal/ws"
)

func main() {
	// 1. Initialize WebSocket Hub
	hub := ws.NewHub()
	go hub.Run()

	// 2. Initialize Whisper Executor
	// Data stored in ./data relative to execution root
	dataDir := "./data"
	whisper := docker.NewWhisperExecutor(dataDir)

	// 3. Define Processing Callback
	processFunc := func(audioPath string) {
		log.Printf("Processing audio chunk: %s", audioPath)

		text, err := whisper.Process(audioPath)
		if err != nil {
			log.Printf("Whisper failed: %v", err)
			return
		}

		if text == "" {
			log.Println("Whisper output empty, skipping broadcast.")
			return
		}

		log.Printf("Transcribed text: %s", text)

		// Broadcast with translation
		hub.BroadcastWithTranslation(text, translate.TranslateText)
	}

	// 4. Initialize Audio Ingestor
	// Rotate every 6 seconds (balance between latency and docker overhead)
	ingestor := audio.NewAudioIngestor(dataDir, 6*time.Second, processFunc)

	// Start Ingestor (consumes hub.AudioChan)
	go ingestor.Start(hub.AudioChan)

	// 5. Setup Server
	mux := http.NewServeMux()

	// Static
	fileServer := http.FileServer(http.Dir("./web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

	// WebSocket Endpoint
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws.ServeWs(hub, w, r)
	})

	// Home
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join("web", "templates", "index.html"))
	})

	log.Println("Real-time Translation Server started on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
