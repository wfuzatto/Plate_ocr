package domain

import "time"

const EventTypePlateDetectedV1 = "nvr.event.plate.detected.v1"

type BBox struct {
	X1 float64 `json:"x1"`
	Y1 float64 `json:"y1"`
	X2 float64 `json:"x2"`
	Y2 float64 `json:"y2"`
}

func (b BBox) Valid() bool {
	return b.X1 >= 0 && b.Y1 >= 0 && b.X2 <= 1 && b.Y2 <= 1 && b.X2 > b.X1 && b.Y2 > b.Y1
}

func (b BBox) IoU(o BBox) float64 {
	x1 := maxf(b.X1, o.X1)
	y1 := maxf(b.Y1, o.Y1)
	x2 := minf(b.X2, o.X2)
	y2 := minf(b.Y2, o.Y2)
	w, h := x2-x1, y2-y1
	if w <= 0 || h <= 0 { return 0 }
	inter := w * h
	a := (b.X2 - b.X1) * (b.Y2 - b.Y1)
	c := (o.X2 - o.X1) * (o.Y2 - o.Y1)
	return inter / (a + c - inter)
}

func (b BBox) CenterInside(o BBox) bool {
	cx, cy := (b.X1+b.X2)/2, (b.Y1+b.Y2)/2
	return cx >= o.X1 && cx <= o.X2 && cy >= o.Y1 && cy <= o.Y2
}

func maxf(a, b float64) float64 { if a > b { return a }; return b }
func minf(a, b float64) float64 { if a < b { return a }; return b }

type RawOCRCandidate struct {
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence"`
}

type Candidate struct {
	RawText        string   `json:"raw_text"`
	NormalizedText string   `json:"normalized_text"`
	Format         string   `json:"format"`
	Confidence     float64  `json:"confidence"`
	Corrections    []string `json:"corrections,omitempty"`
}

type Observation struct {
	CameraID           string            `json:"camera_id"`
	ObservedAt         time.Time         `json:"observed_at"`
	DetectorConfidence float64           `json:"detector_confidence"`
	BBox               BBox              `json:"bbox"`
	OCR                []RawOCRCandidate `json:"ocr"`
	Lane               string            `json:"lane,omitempty"`
	Direction          string            `json:"direction,omitempty"`
	EvidenceJPEG       []byte            `json:"-"`
}

type NormalizedObservation struct {
	CameraID           string
	ObservedAt         time.Time
	DetectorConfidence float64
	BBox               BBox
	Candidates         []Candidate
	Lane               string
	Direction          string
	EvidenceJPEG       []byte
}

type PlateEvent struct {
	EventID            string      `json:"event_id"`
	EventType          string      `json:"event_type"`
	SchemaVersion      string      `json:"schema_version"`
	CameraID           string      `json:"camera_id"`
	ObservedAt         time.Time   `json:"observed_at"`
	PluginID           string      `json:"plugin_id"`
	PluginVersion      string      `json:"plugin_version"`
	Confidence         float64     `json:"confidence"`
	DedupeKey          string      `json:"dedupe_key"`
	TrackID            string      `json:"track_id"`
	RawText            string      `json:"raw_text"`
	NormalizedText     string      `json:"normalized_text"`
	Format             string      `json:"format"`
	Candidates         []Candidate `json:"candidates"`
	DetectorConfidence float64     `json:"detector_confidence"`
	BBox               BBox        `json:"bbox"`
	Lane               string      `json:"lane,omitempty"`
	Direction          string      `json:"direction,omitempty"`
	SnapshotRef        string      `json:"snapshot_ref,omitempty"`
	EvidenceJPEG       []byte      `json:"-"`
}
