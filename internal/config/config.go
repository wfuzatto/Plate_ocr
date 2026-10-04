package config

import (
	"encoding/json"
	"errors"
	"os"
	"time"
)

type Config struct {
	TrackTTL                 time.Duration
	DedupeTTL                time.Duration
	MinObservations          int
	MinEventConfidence       float64
	MaxCandidatesPerRead     int
	MaxActiveTracksPerCamera int
}

type diskConfig struct {
	TrackTTLMS               int64   `json:"track_ttl_ms"`
	DedupeTTLMS              int64   `json:"dedupe_ttl_ms"`
	MinObservations          int     `json:"min_observations"`
	MinEventConfidence       float64 `json:"min_event_confidence"`
	MaxCandidatesPerRead     int     `json:"max_candidates_per_read"`
	MaxActiveTracksPerCamera int     `json:"max_active_tracks_per_camera"`
}

func Default() Config {
	return Config{
		TrackTTL: 1200 * time.Millisecond, DedupeTTL: 5 * time.Second,
		MinObservations: 2, MinEventConfidence: .70, MaxCandidatesPerRead: 8,
		MaxActiveTracksPerCamera: 128,
	}
}

func Load(path string) (Config, error) {
	if path == "" { return Default(), nil }
	b, err := os.ReadFile(path)
	if err != nil { return Config{}, err }
	var d diskConfig
	if err := json.Unmarshal(b, &d); err != nil { return Config{}, err }
	c := Default()
	if d.TrackTTLMS > 0 { c.TrackTTL = time.Duration(d.TrackTTLMS) * time.Millisecond }
	if d.DedupeTTLMS > 0 { c.DedupeTTL = time.Duration(d.DedupeTTLMS) * time.Millisecond }
	if d.MinObservations > 0 { c.MinObservations = d.MinObservations }
	if d.MinEventConfidence > 0 { c.MinEventConfidence = d.MinEventConfidence }
	if d.MaxCandidatesPerRead > 0 { c.MaxCandidatesPerRead = d.MaxCandidatesPerRead }
	if d.MaxActiveTracksPerCamera > 0 { c.MaxActiveTracksPerCamera = d.MaxActiveTracksPerCamera }
	if c.MinEventConfidence < 0 || c.MinEventConfidence > 1 {
		return Config{}, errors.New("min_event_confidence must be between 0 and 1")
	}
	return c, nil
}
