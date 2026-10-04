package inference

import (
	"context"
	"image"
	"time"

	"github.com/wfuzatto/Plate_ocr/internal/domain"
)

type PixelFormat string

const (
	PixelNV12  PixelFormat = "NV12"
	PixelRGB24 PixelFormat = "RGB24"
	PixelBGR24 PixelFormat = "BGR24"
	PixelJPEG  PixelFormat = "JPEG"
)

type Frame struct {
	FrameID    string
	CameraID   string
	ObservedAt time.Time
	Width      int
	Height     int
	Stride     int
	Format     PixelFormat
	Data       []byte
	Image      image.Image
	ROI        *domain.BBox
	Lane       string
	Direction  string
}

type Detection struct {
	BBox       domain.BBox
	Confidence float64
	Lane       string
	Direction  string
}

type Detector interface {
	Detect(context.Context, Frame) ([]Detection, error)
}

type Recognizer interface {
	Recognize(context.Context, Frame, Detection) ([]domain.RawOCRCandidate, error)
}

type ProviderInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Device      string `json:"device"`
	ModelID     string `json:"model_id"`
	ModelSHA256 string `json:"model_sha256,omitempty"`
}
