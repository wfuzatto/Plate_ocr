package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wfuzatto/Plate_ocr/internal/domain"
)

type CameraConfig struct {
	Enabled   *bool        `json:"enabled,omitempty"`
	ROI       *domain.BBox `json:"roi,omitempty"`
	Lane      string       `json:"lane,omitempty"`
	Direction string       `json:"direction,omitempty"`
}

type Config struct {
	TrackTTL                 time.Duration
	DedupeTTL                time.Duration
	MinObservations          int
	MinEventConfidence       float64
	MaxCandidatesPerRead     int
	MaxActiveTracksPerCamera int

	NVRBaseURL      string
	PluginTokenFile string
	ListenAddress   string
	PollFPS         float64
	HTTPTimeout     time.Duration
	SpoolDir        string
	MaxParallel     int
	EvidenceEnabled bool
	CameraIDs       []string
	Cameras         map[string]CameraConfig

	DetectorThreshold   float64
	RecognizerThreshold float64
	MaxDetections       int
	MinPlateWidthPX     int
	MinPlateHeightPX    int
}

type diskConfig struct {
	TrackTTLMS               int64   `json:"track_ttl_ms"`
	DedupeTTLMS              int64   `json:"dedupe_ttl_ms"`
	MinObservations          int     `json:"min_observations"`
	MinEventConfidence       float64 `json:"min_event_confidence"`
	MaxCandidatesPerRead     int     `json:"max_candidates_per_read"`
	MaxActiveTracksPerCamera int     `json:"max_active_tracks_per_camera"`

	NVRBaseURL      string   `json:"nvr_base_url"`
	PluginTokenFile string   `json:"plugin_token_file"`
	ListenAddress   string   `json:"listen"`
	PollFPS         float64  `json:"poll_fps"`
	HTTPTimeoutMS   int64    `json:"http_timeout_ms"`
	SpoolDir        string   `json:"spool_dir"`
	MaxParallel     int      `json:"max_parallel"`
	EvidenceEnabled *bool    `json:"evidence_enabled"`
	CameraIDs       []string `json:"camera_ids"`
	Cameras         map[string]CameraConfig `json:"cameras"`

	DetectorThreshold   float64 `json:"detector_threshold"`
	RecognizerThreshold float64 `json:"recognizer_threshold"`
	MaxDetections       int     `json:"max_detections"`
	MinPlateWidthPX     int     `json:"min_plate_width_px"`
	MinPlateHeightPX    int     `json:"min_plate_height_px"`
}

func Default() Config {
	return Config{
		TrackTTL: 1200 * time.Millisecond,
		DedupeTTL: 5 * time.Second,
		MinObservations: 2,
		MinEventConfidence: .62,
		MaxCandidatesPerRead: 8,
		MaxActiveTracksPerCamera: 64,
		NVRBaseURL: "http://127.0.0.1:8080",
		PluginTokenFile: "./data/plugin.token",
		ListenAddress: "127.0.0.1:8091",
		PollFPS: 3,
		HTTPTimeout: 5 * time.Second,
		SpoolDir: "./data/spool",
		MaxParallel: 8,
		EvidenceEnabled: true,
		Cameras: make(map[string]CameraConfig),
		DetectorThreshold: .36,
		RecognizerThreshold: .48,
		MaxDetections: 8,
		MinPlateWidthPX: 70,
		MinPlateHeightPX: 20,
	}
}

func Load(path string) (Config, error) {
	c := Default()
	if strings.TrimSpace(path) == "" { return applyEnvAndValidate(c) }
	b, err := os.ReadFile(path)
	if err != nil { return Config{}, err }
	var d diskConfig
	if err := json.Unmarshal(b, &d); err != nil { return Config{}, err }

	if d.TrackTTLMS > 0 { c.TrackTTL = time.Duration(d.TrackTTLMS) * time.Millisecond }
	if d.DedupeTTLMS > 0 { c.DedupeTTL = time.Duration(d.DedupeTTLMS) * time.Millisecond }
	if d.MinObservations > 0 { c.MinObservations = d.MinObservations }
	if d.MinEventConfidence > 0 { c.MinEventConfidence = d.MinEventConfidence }
	if d.MaxCandidatesPerRead > 0 { c.MaxCandidatesPerRead = d.MaxCandidatesPerRead }
	if d.MaxActiveTracksPerCamera > 0 { c.MaxActiveTracksPerCamera = d.MaxActiveTracksPerCamera }
	if strings.TrimSpace(d.NVRBaseURL) != "" { c.NVRBaseURL = strings.TrimRight(strings.TrimSpace(d.NVRBaseURL), "/") }
	if strings.TrimSpace(d.PluginTokenFile) != "" { c.PluginTokenFile = d.PluginTokenFile }
	if strings.TrimSpace(d.ListenAddress) != "" { c.ListenAddress = d.ListenAddress }
	if d.PollFPS > 0 { c.PollFPS = d.PollFPS }
	if d.HTTPTimeoutMS > 0 { c.HTTPTimeout = time.Duration(d.HTTPTimeoutMS) * time.Millisecond }
	if strings.TrimSpace(d.SpoolDir) != "" { c.SpoolDir = d.SpoolDir }
	if d.MaxParallel > 0 { c.MaxParallel = d.MaxParallel }
	if d.EvidenceEnabled != nil { c.EvidenceEnabled = *d.EvidenceEnabled }
	if d.CameraIDs != nil { c.CameraIDs = append([]string(nil), d.CameraIDs...) }
	if d.Cameras != nil { c.Cameras = d.Cameras }
	if d.DetectorThreshold > 0 { c.DetectorThreshold = d.DetectorThreshold }
	if d.RecognizerThreshold > 0 { c.RecognizerThreshold = d.RecognizerThreshold }
	if d.MaxDetections > 0 { c.MaxDetections = d.MaxDetections }
	if d.MinPlateWidthPX > 0 { c.MinPlateWidthPX = d.MinPlateWidthPX }
	if d.MinPlateHeightPX > 0 { c.MinPlateHeightPX = d.MinPlateHeightPX }
	return applyEnvAndValidate(c)
}

func applyEnvAndValidate(c Config) (Config, error) {
	if v := strings.TrimSpace(os.Getenv("PLATE_OCR_NVR_URL")); v != "" { c.NVRBaseURL = strings.TrimRight(v, "/") }
	if v := strings.TrimSpace(os.Getenv("PLATE_OCR_TOKEN_FILE")); v != "" { c.PluginTokenFile = v }
	if v := strings.TrimSpace(os.Getenv("PLATE_OCR_LISTEN")); v != "" { c.ListenAddress = v }
	if v := strings.TrimSpace(os.Getenv("PLATE_OCR_SPOOL_DIR")); v != "" { c.SpoolDir = v }

	if c.MinEventConfidence < 0 || c.MinEventConfidence > 1 { return Config{}, errors.New("min_event_confidence must be between 0 and 1") }
	if c.DetectorThreshold < 0 || c.DetectorThreshold > 1 { return Config{}, errors.New("detector_threshold must be between 0 and 1") }
	if c.RecognizerThreshold < 0 || c.RecognizerThreshold > 1 { return Config{}, errors.New("recognizer_threshold must be between 0 and 1") }
	if c.PollFPS <= 0 || c.PollFPS > 15 { return Config{}, errors.New("poll_fps must be > 0 and <= 15") }
	if c.MaxParallel < 1 || c.MaxParallel > 128 { return Config{}, errors.New("max_parallel must be between 1 and 128") }
	if c.HTTPTimeout <= 0 { return Config{}, errors.New("http timeout must be > 0") }
	if c.MaxDetections < 1 || c.MaxDetections > 64 { return Config{}, errors.New("max_detections must be between 1 and 64") }
	for id, cam := range c.Cameras {
		if cam.ROI != nil && !cam.ROI.Valid() { return Config{}, fmt.Errorf("camera %s has invalid roi", id) }
	}
	if err := os.MkdirAll(c.SpoolDir, 0o750); err != nil { return Config{}, fmt.Errorf("create spool dir: %w", err) }
	c.PluginTokenFile = filepath.Clean(c.PluginTokenFile)
	return c, nil
}
