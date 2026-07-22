package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "media-inspector/proto/inspectorpb"
)

// target pairs the label to display for a file (e.g. the original
// Windows path the user typed) with the actual path to send the
// server (e.g. the path as mounted inside the container).
type target struct {
	label string
	path  string
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <path-to-media-file> [more-paths...]\n", os.Args[0])
		os.Exit(1)
	}

	targets := make([]target, 0, len(os.Args)-1)
	paths := make([]string, 0, len(os.Args)-1)
	for _, arg := range os.Args[1:] {
		if label, path, found := strings.Cut(arg, "|||"); found {
			targets = append(targets, target{label: label, path: path})
			paths = append(paths, path)
		} else {
			targets = append(targets, target{label: arg, path: arg})
			paths = append(paths, arg)
		}
	}

	conn, err := grpc.NewClient("localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to connect to server: %v", err)
	}
	defer conn.Close()

	client := pb.NewMediaInspectorClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.Inspect(ctx, &pb.InspectRequest{FilePaths: paths})
	if err != nil {
		log.Fatalf("Inspect failed: %v", err)
	}

	results := resp.GetResults()
	for i, t := range targets {
		fmt.Printf("=== %s ===\n", t.label)

		if i >= len(results) {
			fmt.Printf("Error: no result returned for this file\n\n")
			continue
		}

		result := results[i]
		if result.GetError() != "" {
			fmt.Printf("Error: %s\n\n", result.GetError())
			continue
		}
		fmt.Print(formatResult(result))
		fmt.Println()
	}
}

func formatResult(result *pb.FileResult) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%-15s: %s\n", "Container", containerLabel(result.GetContainer()))
	fmt.Fprintf(&b, "%-15s: %.3f seconds\n", "Duration", result.GetDurationSeconds())

	for _, s := range result.GetStreams() {
		switch {
		case s.GetVideo() != nil:
			v := s.GetVideo()
			fmt.Fprintf(&b, "%-15s: %s\n", "Video Codec", codecLabel(s.GetCodec()))
			if v.GetWidth() > 0 || v.GetHeight() > 0 {
				fmt.Fprintf(&b, "%-15s: %d x %d\n", "Resolution", v.GetWidth(), v.GetHeight())
			}
			if v.GetFps() != "" {
				fmt.Fprintf(&b, "%-15s: %s\n", "FPS", fpsLabel(v.GetFps()))
			}
			if s.GetBitrate() > 0 {
				fmt.Fprintf(&b, "%-15s: %d bps\n", "Video Bitrate", s.GetBitrate())
			}
		case s.GetAudio() != nil:
			a := s.GetAudio()
			fmt.Fprintf(&b, "%-15s: %s\n", "Audio Codec", codecLabel(s.GetCodec()))
			if a.GetChannels() > 0 {
				fmt.Fprintf(&b, "%-15s: %d\n", "Channels", a.GetChannels())
			}
			if a.GetSampleRate() > 0 {
				fmt.Fprintf(&b, "%-15s: %d Hz\n", "Sample Rate", a.GetSampleRate())
			}
			if s.GetBitrate() > 0 {
				fmt.Fprintf(&b, "%-15s: %d bps\n", "Audio Bitrate", s.GetBitrate())
			}
		default:
			fmt.Fprintf(&b, "%-15s: %s\n", "Stream", codecLabel(s.GetCodec()))
		}
	}

	return b.String()
}

func containerLabel(caps string) string {
	switch {
	case strings.Contains(caps, "video/quicktime"):
		return "MP4/QuickTime"
	case strings.Contains(caps, "video/x-matroska"):
		return "MKV"
	case strings.Contains(caps, "video/mpegts"):
		return "MPEG-TS"
	case strings.Contains(caps, "video/x-msvideo"):
		return "AVI"
	case strings.Contains(caps, "application/x-id3"):
		return "MP3 (ID3)"
	default:
		return caps
	}
}

func codecLabel(mediaType string) string {
	labels := map[string]string{
		"video/x-h264": "H.264",
		"video/x-h265": "H.265",
		"video/x-vp8":  "VP8",
		"video/x-vp9":  "VP9",
		"audio/mpeg":   "AAC/MP3",
		"audio/x-opus": "Opus",
		"audio/x-flac": "FLAC",
	}
	if label, ok := labels[mediaType]; ok {
		return label
	}
	return mediaType
}

func fpsLabel(fps string) string {
	var n, d int
	if _, err := fmt.Sscanf(fps, "%d/%d", &n, &d); err == nil && d == 1 {
		return fmt.Sprintf("%d", n)
	}
	return fps
}
