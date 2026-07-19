// Package inspector wraps the C GStreamer-backed media inspector via cgo.
package inspector

/*
#cgo pkg-config: gstreamer-1.0 gstreamer-pbutils-1.0
#cgo CFLAGS: -I${SRCDIR}/../../c
#include <stdlib.h>
#include "inspector.h"
#include "inspector.c"
*/
import "C"

import (
	"encoding/json"
	"errors"
	"fmt"
	"unsafe"
)

// ErrInspectFailed is returned when the C layer could not inspect the file.
var ErrInspectFailed = errors.New("inspector: could not inspect media file")

// Stream describes one video/audio track found in the media file.
type Stream struct {
	Type       string `json:"type"`
	Codec      string `json:"codec"`
	Width      uint32 `json:"width,omitempty"`
	Height     uint32 `json:"height,omitempty"`
	FPS        string `json:"fps,omitempty"`
	Channels   uint32 `json:"channels,omitempty"`
	SampleRate uint32 `json:"sample_rate,omitempty"`
		Bitrate    uint32 `json:"bitrate,omitempty"`

}

// MediaInfo is the parsed result of inspecting a media file. Error is
// populated instead of Container/Streams when GStreamer could not
// inspect the file — callers should check Error first.
type MediaInfo struct {
	Container       string   `json:"container"`
	DurationSeconds float64  `json:"duration_seconds"`
	Streams         []Stream `json:"streams"`
	Error           string   `json:"error,omitempty"`
}

// Inspect opens filepath with GStreamer (via cgo) and returns a parsed
// description of it.
func Inspect(filepath string) (*MediaInfo, error) {
	if filepath == "" {
		return nil, fmt.Errorf("inspector: empty file path")
	}

	cPath := C.CString(filepath)
	defer C.free(unsafe.Pointer(cPath))

	cResult := C.inspect_media(cPath)
	if cResult == nil {
		return nil, fmt.Errorf("%w: %s", ErrInspectFailed, filepath)
	}
	defer C.inspect_media_free(cResult)

	goJSON := C.GoString(cResult)

	var info MediaInfo
	if err := json.Unmarshal([]byte(goJSON), &info); err != nil {
		return nil, fmt.Errorf("inspector: parsing result for %s: %w", filepath, err)
	}
	if info.Error != "" {
		return nil, fmt.Errorf("%w: %s: %s", ErrInspectFailed, filepath, info.Error)
	}
	return &info, nil
}