package docker

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

type WhisperExecutor struct {
	DataDirAbs string
}

func NewWhisperExecutor(dataDir string) *WhisperExecutor {
	abs, err := filepath.Abs(dataDir)
	if err != nil {
		log.Fatalf("Failed to get absolute path for data dir: %v", err)
	}
	return &WhisperExecutor{DataDirAbs: abs}
}

func (w *WhisperExecutor) Process(audioFilePath string) (string, error) {
	// audioFilePath is the absolute path on host.
	// We need the filename relative to the mounted volume for the docker command.
	fileName := filepath.Base(audioFilePath)

	// Command:
	// docker run --rm --gpus all -v [DataDir]:/app whisper-gx10 [fileName] --model medium --language Chinese --output_dir /app
	// Remove -d to wait for completion.
	// Ensure --name is unique or omitted (omitted is safer for concurrent runs, though we are sequential per ingestor).

	cmd := exec.Command("docker", "run", "--rm", "--gpus", "all",
		"-v", fmt.Sprintf("%s:/app", w.DataDirAbs),
		"whisper-gx10",
		fileName,
		"--model", "medium",
		"--language", "Chinese",
		"--output_dir", "/app",
	)

	log.Printf("Executing Docker command: %s", cmd.String())

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Docker execution failed: %s\nOutput: %s", err, string(output))
		return "", err
	}

	// Assuming Whisper outputs a .txt file with the same name (or slightly different).
	// Standard whisper usually does [filename].txt
	// If input is chunk_123.mp3, output is chunk_123.txt (or chunk_123.mp3.txt depending on version).
	// Let's guess it replaces extension or appends.
	// "whisper-gx10" might be a wrapper.
	// Let's try reading [filename with ext removed].txt first, then [filename].txt.

	txtPath := audioFilePath[:len(audioFilePath)-len(filepath.Ext(audioFilePath))] + ".txt"
	content, err := os.ReadFile(txtPath)
	if err != nil {
		// Try appending .txt
		txtPath2 := audioFilePath + ".txt"
		content2, err2 := os.ReadFile(txtPath2)
		if err2 != nil {
			log.Printf("Could not find output text file. Checked %s and %s", txtPath, txtPath2)
			return "", fmt.Errorf("text output not found")
		}
		content = content2
	}
	return string(content), nil
}
