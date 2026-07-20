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
func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <path-to-media-file>\n", os.Args[0])
		os.Exit(1)
	}
	filePath := os.Args[1]
	conn, err := grpc.NewClient("localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to connect to server: %v", err)
	}
	defer conn.Close()
	client := pb.NewMediaInspectorClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := client.Inspect(ctx, &pb.InspectRequest{FilePath: filePath})
	if err != nil {
		log.Fatalf("Inspect failed: %v", err)
	}
fmt.Printf("%-15s: %s\n", "Container", containerLabel(resp.GetContainer()))
	fmt.Printf("%-15s: %.3f seconds\n", "Duration", resp.GetDurationSeconds())
	for _, s := range resp.GetStreams() {
		switch s.GetType() {
		case "video":
			fmt.Printf("%-15s: %s\n", "Video Codec", codecLabel(s.GetCodec()))
			if s.GetWidth() > 0 || s.GetHeight() > 0 {
				fmt.Printf("%-15s: %d x %d\n", "Resolution", s.GetWidth(), s.GetHeight())
			}
			if s.GetFps() != "" {
				fmt.Printf("%-15s: %s\n", "FPS", fpsLabel(s.GetFps()))
			}
			if s.GetBitrate() > 0 {
				fmt.Printf("%-15s: %d bps\n", "Video Bitrate", s.GetBitrate())
			}
		case "audio":
			fmt.Printf("%-15s: %s\n", "Audio Codec", codecLabel(s.GetCodec()))
			if s.GetChannels() > 0 {
				fmt.Printf("%-15s: %d\n", "Channels", s.GetChannels())
			}
			if s.GetSampleRate() > 0 {
				fmt.Printf("%-15s: %d Hz\n", "Sample Rate", s.GetSampleRate())
			}
			if s.GetBitrate() > 0 {
				fmt.Printf("%-15s: %d bps\n", "Audio Bitrate", s.GetBitrate())
			}
		default:
			fmt.Printf("%-15s: %s (%s)\n", "Stream", s.GetType(), s.GetCodec())
		}
	}
}
// codecLabel turns a raw GStreamer media-type string (e.g.
// "video/x-h264") into a short human-readable label ("H.264"),
// falling back to the raw string for anything not in the table.
// containerLabel turns a raw GStreamer container caps string into a
// short, familiar format name, falling back to the raw string for
// anything not in the table.
func containerLabel(caps string) string {
	switch {
	case strings.Contains(caps, "video/quicktime"):
		return "MP4"
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
// fpsLabel converts a "30/1"-style fraction into a plain number when
// the denominator is 1, and leaves genuine fractions (like "24000/1001")
// as-is, since collapsing those would lose precision.
func fpsLabel(fps string) string {
	var n, d int
	if _, err := fmt.Sscanf(fps, "%d/%d", &n, &d); err == nil && d == 1 {
		return fmt.Sprintf("%d", n)
	}
	return fps
}