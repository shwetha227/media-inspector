package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "media-inspector/proto/inspectorpb"
)

type target struct {
	label string // what to print
	path  string // what to send to the server
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <path-to-media-file> [more-paths...]\n", os.Args[0])
		os.Exit(1)
	}

	targets := make([]target, 0, len(os.Args)-1)
	for _, arg := range os.Args[1:] {
		if label, path, found := strings.Cut(arg, "|||"); found {
			targets = append(targets, target{label: label, path: path})
		} else {
			targets = append(targets, target{label: arg, path: arg})
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

	var wg sync.WaitGroup
	var printMu sync.Mutex

	for _, t := range targets {
		wg.Add(1)
		go func(t target) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			resp, err := client.Inspect(ctx, &pb.InspectRequest{FilePath: t.path})

			printMu.Lock()
			defer printMu.Unlock()

			if err != nil {
				fmt.Printf("=== %s ===\nError: Inspect failed: %v\n\n", t.label, err)
				return
			}
			if resp.GetError() != "" {
				fmt.Printf("=== %s ===\nError: Inspect failed: %s\n\n", t.label, resp.GetError())
				return
			}

			fmt.Printf("=== %s ===\n%s\n", t.label, formatResult(resp))
		}(t)
	}

	wg.Wait()
}

func formatResult(resp *pb.InspectResponse) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%-15s: %s\n", "Container", containerLabel(resp.GetContainer()))
	fmt.Fprintf(&b, "%-15s: %.3f seconds\n", "Duration", resp.GetDurationSeconds())

	for _, s := range resp.GetStreams() {
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