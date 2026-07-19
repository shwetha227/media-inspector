package main

import (
	"context"
	"net"
	"path/filepath"
	"runtime"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	pb "media-inspector/proto/inspectorpb"
)

const bufSize = 1024 * 1024

// testdataDir resolves testdata/ relative to this test file, regardless
// of the working directory `go test` is invoked from.
func testdataDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "testdata")
}

// newTestClient starts a gRPC server on an in-memory bufconn listener and
// returns a connected client. The server and connection are closed
// automatically via t.Cleanup.
func newTestClient(t *testing.T) pb.MediaInspectorClient {
	t.Helper()

	lis := bufconn.Listen(bufSize)
	grpcServer := grpc.NewServer()
	pb.RegisterMediaInspectorServer(grpcServer, &mediaInspectorServer{})

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			t.Logf("bufconn server exited: %v", err)
		}
	}()
	t.Cleanup(grpcServer.Stop)

	dialer := func(ctx context.Context, _ string) (net.Conn, error) {
		return lis.DialContext(ctx)
	}

	conn, err := grpc.NewClient("passthrough:///bufconn",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("failed to dial bufconn: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	return pb.NewMediaInspectorClient(conn)
}

func TestServer_Inspect_HappyPath(t *testing.T) {
	client := newTestClient(t)
	path := filepath.Join(testdataDir(t), "sample.mp4")

	resp, err := client.Inspect(context.Background(), &pb.InspectRequest{FilePath: path})
	if err != nil {
		t.Fatalf("Inspect returned unexpected error: %v", err)
	}
	if resp.Container == "" {
		t.Error("expected a non-empty container")
	}
	if len(resp.Streams) == 0 {
		t.Fatal("expected at least one stream")
	}
}

func TestServer_Inspect_EmptyPath(t *testing.T) {
	client := newTestClient(t)

	_, err := client.Inspect(context.Background(), &pb.InspectRequest{FilePath: ""})
	if err == nil {
		t.Fatal("expected an error for empty file_path, got nil")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", status.Code(err))
	}
}

func TestServer_Inspect_NonExistentPath(t *testing.T) {
	client := newTestClient(t)
	path := filepath.Join(testdataDir(t), "does-not-exist.mp4")

	_, err := client.Inspect(context.Background(), &pb.InspectRequest{FilePath: path})
	if err == nil {
		t.Fatal("expected an error for a non-existent path, got nil")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", status.Code(err))
	}
}

func TestServer_Inspect_NotMediaFile(t *testing.T) {
	client := newTestClient(t)
	path := filepath.Join(testdataDir(t), "not_media.txt")

	_, err := client.Inspect(context.Background(), &pb.InspectRequest{FilePath: path})
	if err == nil {
		t.Fatal("expected an error for a non-media file, got nil")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", status.Code(err))
	}
}
