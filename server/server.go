package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net"
	"os"

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
	path := req.GetFilePath()
	if path == "" {
		return nil, status.Error(codes.InvalidArgument, "file_path must not be empty")
	}

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, status.Errorf(codes.InvalidArgument, "file not found: %s", path)
		}
		return nil, status.Errorf(codes.InvalidArgument, "cannot access file: %v", err)
	}

	info, err := inspector.Inspect(path)
	if err != nil {
		if errors.Is(err, inspector.ErrInspectFailed) {
			return nil, status.Errorf(codes.InvalidArgument, "could not inspect file: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "inspection failed: %v", err)
	}

	resp := &pb.InspectResponse{
		Container:       info.Container,
		DurationSeconds: info.DurationSeconds,
	}
	for _, s := range info.Streams {
		resp.Streams = append(resp.Streams, &pb.Stream{
			Type:       s.Type,
			Codec:      s.Codec,
			Width:      s.Width,
			Height:     s.Height,
			Fps:        s.FPS,
			Channels:   s.Channels,
			SampleRate: s.SampleRate,
						Bitrate:    s.Bitrate,

		})
	}
	return resp, nil
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