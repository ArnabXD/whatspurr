package main

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os/exec"
	"time"

	_ "golang.org/x/image/webp"
)

// detectImageDimensions reads the image header to get width/height.
// Returns (0, 0) if detection fails — callers should treat that as "unknown".
func detectImageDimensions(data []byte) (width, height uint32) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0, 0
	}
	return uint32(cfg.Width), uint32(cfg.Height)
}

type ffprobeOutput struct {
	Streams []struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"streams"`
}

// detectVideoDimensions shells out to ffprobe to get width/height.
// Returns (0, 0) if ffprobe is not installed or detection fails.
func detectVideoDimensions(data []byte) (width, height uint32) {
	ffprobe, err := exec.LookPath("ffprobe")
	if err != nil {
		return 0, 0
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, ffprobe,
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		"-select_streams", "v:0",
		"-i", "pipe:0",
	)
	cmd.Stdin = bytes.NewReader(data)
	out, err := cmd.Output()
	if err != nil {
		return 0, 0
	}

	var result ffprobeOutput
	if err := json.Unmarshal(out, &result); err != nil || len(result.Streams) == 0 {
		return 0, 0
	}

	return uint32(result.Streams[0].Width), uint32(result.Streams[0].Height)
}
