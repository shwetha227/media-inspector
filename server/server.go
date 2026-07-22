package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
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
			return &pb.InspectResponse{
				Error: fmt.Sprintf("file not found: %s", path),
			}, nil
		}
		return &pb.InspectResponse{
			Error: fmt.Sprintf("cannot access file: %v", err),
		}, nil
	}

	info, err := inspector.Inspect(path)
	if err != nil {
		if errors.Is(err, inspector.ErrInspectFailed) {
			return &pb.InspectResponse{
				Error: fmt.Sprintf("could not inspect file: %v", err),
			}, nil
		}
		return nil, status.Errorf(codes.Internal, "inspection failed: %v", err)
	}

	resp := &pb.InspectResponse{
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
		resp.Streams = append(resp.Streams, stream)
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