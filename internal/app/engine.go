package app

import (
	"errors"
	"sync/atomic"
	"time"

	"github.com/wfuzatto/Plate_ocr/internal/aggregate"
	"github.com/wfuzatto/Plate_ocr/internal/config"
	"github.com/wfuzatto/Plate_ocr/internal/dedupe"
	"github.com/wfuzatto/Plate_ocr/internal/domain"
	"github.com/wfuzatto/Plate_ocr/internal/normalize"
)

type Metrics struct {
	Received, Rejected, Published, Deduplicated atomic.Uint64
}
type Engine struct {
	cfg config.Config
	agg *aggregate.Manager
	dedupe *dedupe.Cache
	Metrics Metrics
}

func New(cfg config.Config) *Engine {
	return &Engine{
		cfg:cfg,
		agg:aggregate.New(aggregate.Config{
			TrackTTL:cfg.TrackTTL, MinObservations:cfg.MinObservations,
			MinConfidence:cfg.MinEventConfidence, MaxActivePerCamera:cfg.MaxActiveTracksPerCamera,
		}),
		dedupe:dedupe.New(cfg.DedupeTTL),
	}
}

func (e *Engine) Process(o domain.Observation) ([]domain.PlateEvent,error) {
	e.Metrics.Received.Add(1)
	if o.CameraID=="" {
		e.Metrics.Rejected.Add(1)
		return nil, errors.New("camera_id is required")
	}
	if o.ObservedAt.IsZero() { o.ObservedAt=time.Now().UTC() }
	if !o.BBox.Valid() {
		e.Metrics.Rejected.Add(1)
		return nil, errors.New("invalid normalized bbox")
	}
	if len(o.OCR)==0 {
		e.Metrics.Rejected.Add(1)
		return e.flush(o.ObservedAt),nil
	}
	all:=make([]domain.Candidate,0,e.cfg.MaxCandidatesPerRead)
	for _,raw:=range o.OCR {
		for _,c:=range normalize.Candidates(raw.Text,raw.Confidence,e.cfg.MaxCandidatesPerRead) {
			all=append(all,c)
			if len(all)>=e.cfg.MaxCandidatesPerRead { break }
		}
		if len(all)>=e.cfg.MaxCandidatesPerRead { break }
	}
	if len(all)==0 {
		e.Metrics.Rejected.Add(1)
		return e.flush(o.ObservedAt),nil
	}
	e.agg.Observe(domain.NormalizedObservation{
		CameraID:o.CameraID, ObservedAt:o.ObservedAt, DetectorConfidence:o.DetectorConfidence,
		BBox:o.BBox, Candidates:all, Lane:o.Lane, Direction:o.Direction, EvidenceJPEG:o.EvidenceJPEG,
	})
	return e.flush(o.ObservedAt),nil
}

func (e *Engine) Tick(now time.Time) []domain.PlateEvent {
	if now.IsZero(){now=time.Now().UTC()}
	return e.flush(now)
}
func (e *Engine) FlushAll() []domain.PlateEvent { return e.filter(e.agg.FlushAll()) }
func (e *Engine) flush(now time.Time) []domain.PlateEvent { return e.filter(e.agg.FlushExpired(now)) }
func (e *Engine) filter(in []domain.PlateEvent) []domain.PlateEvent {
	out:=in[:0]
	for _,ev:=range in {
		if e.dedupe.Accept(ev.DedupeKey,ev.ObservedAt) {
			out=append(out,ev); e.Metrics.Published.Add(1)
		} else { e.Metrics.Deduplicated.Add(1) }
	}
	return out
}
