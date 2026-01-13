package audio

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

type AudioIngestor struct {
	DataDir      string
	CurrentFile  *os.File
	ProcessFunc  func(filePath string) // Callback when a chunk is ready
	RotationTime time.Duration
}

func NewAudioIngestor(dataDir string, rotationTime time.Duration, processFunc func(string)) *AudioIngestor {
	return &AudioIngestor{
		DataDir:      dataDir,
		RotationTime: rotationTime,
		ProcessFunc:  processFunc,
	}
}

func (ai *AudioIngestor) Start(audioChan <-chan []byte) {
	// Create data dir if not exists
	if _, err := os.Stat(ai.DataDir); os.IsNotExist(err) {
		os.MkdirAll(ai.DataDir, 0755)
	}

	// Open first file
	if err := ai.rotateFile(); err != nil {
		log.Fatalf("Failed to create initial audio file: %v", err)
	}

	ticker := time.NewTicker(ai.RotationTime)
	defer ticker.Stop()

	log.Printf("Audio Ingestor started. Files output to: %s", ai.DataDir)

	for {
		select {
		case chunk := <-audioChan:
			if ai.CurrentFile != nil {
				if _, err := ai.CurrentFile.Write(chunk); err != nil {
					log.Printf("Error writing to audio file: %v", err)
				}
			}
		case <-ticker.C:
			// Time to rotate
			oldPath := ai.CurrentFile.Name()
			if err := ai.rotateFile(); err != nil {
				log.Printf("Error rotating file: %v", err)
				continue
			}
			// Process the old file in a separate goroutine to not block writing
			go ai.ProcessFunc(oldPath)
		}
	}
}

func (ai *AudioIngestor) rotateFile() error {
	// Close existing
	if ai.CurrentFile != nil {
		ai.CurrentFile.Close()
	}

	// Open new
	fileName := fmt.Sprintf("chunk_%d.mp3", time.Now().UnixNano()) // Using .mp3 extension as container expects arbitrary audio, but mp3 is safe
	path := filepath.Join(ai.DataDir, fileName)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	ai.CurrentFile = f
	return nil
}
