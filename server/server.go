package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"media-inspector/internal/inspector"
	pb "media-inspector/proto/inspectorpb"
)

var addr = flag.String("addr", ":50051", "address to listen on")

type mediaInspectorServer struct {
	pb.UnimplementedMediaInspectorServer
}

func (s *mediaInspectorServer) Inspect(ctx context.Context, req *pb.InspectRequest) (*pb.InspectResponse, error) {
	paths := req.GetFilePaths()

	if len(paths) == 0 {
		return nil, status.Error(codes.InvalidArgument, "file_paths must not be empty")
	}

	results := make([]*pb.FileResult, len(paths))

	var wg sync.WaitGroup
	for i, path := range paths {
		wg.Add(1)
		go func(i int, path string) {
			defer wg.Done()
			results[i] = inspectOne(path)
		}(i, path)
	}
	wg.Wait()

	return &pb.InspectResponse{Results: results}, nil
}

func inspectOne(path string) *pb.FileResult {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return &pb.FileResult{
				FilePath: path,
				Error:    fmt.Sprintf("file not found: %s", path),
			}
		}
		return &pb.FileResult{
			FilePath: path,
			Error:    fmt.Sprintf("cannot access file: %v", err),
		}
	}

	info, err := inspector.Inspect(path)
	if err != nil {
		if errors.Is(err, inspector.ErrInspectFailed) {
			return &pb.FileResult{
				FilePath: path,
				Error:    fmt.Sprintf("could not inspect file: %v", err),
			}
		}
		return &pb.FileResult{
			FilePath: path,
			Error:    fmt.Sprintf("inspection failed: %v", err),
		}
	}

	result := &pb.FileResult{
		FilePath:        path,
		Container:       info.Container,
		DurationSeconds: info.DurationSeconds,
	}

	for _, st := range info.Streams {
		stream := &pb.Stream{
			Codec:   st.Codec,
			Bitrate: st.Bitrate,
		}

		switch st.Type {
		case "video":
			stream.Details = &pb.Stream_Video{
				Video: &pb.VideoDetails{
					Width:  st.Width,
					Height: st.Height,
					Fps:    st.FPS,
				},
			}
		case "audio":
			stream.Details = &pb.Stream_Audio{
				Audio: &pb.AudioDetails{
					Channels:   st.Channels,
					SampleRate: st.SampleRate,
				},
			}
		}
		result.Streams = append(result.Streams, stream)
	}

	return result
}

func main() {
	flag.Parse()

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", *addr, err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterMediaInspectorServer(grpcServer, &mediaInspectorServer{})

	log.Printf("media-inspector gRPC server listening on %s", *addr)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}