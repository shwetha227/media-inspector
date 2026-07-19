package main

import (
	"context"
	"fmt"
	"log"
	"os"
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

	fmt.Printf("File:       %s\n", filePath)
	fmt.Printf("Container:  %s\n", resp.GetContainer())
	fmt.Printf("Duration:   %.2fs\n", resp.GetDurationSeconds())
	fmt.Printf("Streams:\n")
	for i, s := range resp.GetStreams() {
		fmt.Printf("  [%d] type=%s codec=%s", i, s.GetType(), s.GetCodec())
		if s.GetWidth() > 0 || s.GetHeight() > 0 {
			fmt.Printf(" resolution=%dx%d", s.GetWidth(), s.GetHeight())
		}
		if s.GetFps() != "" {
			fmt.Printf(" fps=%s", s.GetFps())
		}
		if s.GetChannels() > 0 {
			fmt.Printf(" channels=%d", s.GetChannels())
		}
		if s.GetSampleRate() > 0 {
			fmt.Printf(" sampleRate=%d", s.GetSampleRate())
		}
		if s.GetBitrate() > 0 {
			fmt.Printf(" bitrate=%d", s.GetBitrate())
		}
		fmt.Println()
	}
}
