package aggregate

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/wfuzatto/Plate_ocr/internal/domain"
)

type Config struct {
	TrackTTL time.Duration
	MinObservations int
	MinConfidence float64
	MaxActivePerCamera int
}
type candidateScore struct {
	candidate domain.Candidate
	total float64
	hits int
}
type track struct {
	id string
	camera string
	first, last time.Time
	observations int
	bbox domain.BBox
	detectorBest float64
	lane, direction string
	scores map[string]*candidateScore
	evidence []byte
}
type Manager struct {
	mu sync.Mutex
	cfg Config
	byCamera map[string][]*track
}

func New(cfg Config) *Manager {
	if cfg.MaxActivePerCamera < 1 { cfg.MaxActivePerCamera = 64 }
	return &Manager{cfg:cfg, byCamera:make(map[string][]*track)}
}

func (m *Manager) Observe(o domain.NormalizedObservation) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tracks := m.byCamera[o.CameraID]
	var best *track
	bestScore := -1.0
	for _, t := range tracks {
		if o.ObservedAt.Sub(t.last) > m.cfg.TrackTTL || o.ObservedAt.Before(t.last.Add(-250*time.Millisecond)) { continue }
		s := matchScore(t,o)
		if s > bestScore { best, bestScore = t, s }
	}
	if best == nil || bestScore < .30 {
		best = &track{
			id:trackID(o), camera:o.CameraID, first:o.ObservedAt, last:o.ObservedAt,
			bbox:o.BBox, lane:o.Lane, direction:o.Direction, scores:make(map[string]*candidateScore),
		}
		tracks = append(tracks,best)
		if len(tracks) > m.cfg.MaxActivePerCamera {
			sort.Slice(tracks, func(i,j int) bool { return tracks[i].last.Before(tracks[j].last) })
			tracks = tracks[len(tracks)-m.cfg.MaxActivePerCamera:]
		}
		m.byCamera[o.CameraID] = tracks
	}
	best.last = o.ObservedAt
	best.observations++
	if o.DetectorConfidence >= best.detectorBest {
		best.detectorBest = o.DetectorConfidence
		best.bbox = o.BBox
		if o.Lane != "" { best.lane = o.Lane }
		if o.Direction != "" { best.direction = o.Direction }
		if len(o.EvidenceJPEG)>0 { best.evidence=append(best.evidence[:0],o.EvidenceJPEG...) }
	} else if len(best.evidence)==0 && len(o.EvidenceJPEG)>0 {
		best.evidence=append([]byte(nil),o.EvidenceJPEG...)
	}
	for _, c := range o.Candidates {
		key := c.NormalizedText + "|" + c.Format
		cs := best.scores[key]
		if cs == nil {
			cs = &candidateScore{candidate:c}
			best.scores[key] = cs
		}
		cs.total += c.Confidence * clamp(o.DetectorConfidence)
		cs.hits++
		if c.Confidence > cs.candidate.Confidence { cs.candidate = c }
	}
}

func (m *Manager) FlushExpired(now time.Time) []domain.PlateEvent {
	m.mu.Lock(); defer m.mu.Unlock()
	var out []domain.PlateEvent
	for camera, tracks := range m.byCamera {
		keep := tracks[:0]
		for _, t := range tracks {
			if now.Sub(t.last) >= m.cfg.TrackTTL {
				if e, ok := m.finalize(t); ok { out = append(out,e) }
			} else { keep = append(keep,t) }
		}
		if len(keep)==0 { delete(m.byCamera,camera) } else { m.byCamera[camera]=keep }
	}
	return out
}

func (m *Manager) FlushAll() []domain.PlateEvent {
	m.mu.Lock(); defer m.mu.Unlock()
	var out []domain.PlateEvent
	for camera, tracks := range m.byCamera {
		for _, t := range tracks { if e, ok := m.finalize(t); ok { out = append(out,e) } }
		delete(m.byCamera,camera)
	}
	return out
}

func (m *Manager) finalize(t *track) (domain.PlateEvent,bool) {
	if t.observations < m.cfg.MinObservations || len(t.scores)==0 { return domain.PlateEvent{},false }
	arr := make([]*candidateScore,0,len(t.scores))
	for _, s := range t.scores { arr = append(arr,s) }
	sort.Slice(arr, func(i,j int) bool { return arr[i].total > arr[j].total })
	best := arr[0]
	conf := best.total / float64(maxi(1,t.observations))
	if conf > 1 { conf = 1 }
	if conf < m.cfg.MinConfidence { return domain.PlateEvent{},false }
	cands := make([]domain.Candidate,0,minInt(8,len(arr)))
	for i, s := range arr {
		if i >= 8 { break }
		c := s.candidate
		c.Confidence = clamp(s.total/float64(maxi(1,t.observations)))
		cands = append(cands,c)
	}
	c := best.candidate
	dedupe := t.camera + ":" + c.NormalizedText
	return domain.PlateEvent{
		EventID:eventID(t.id,c.NormalizedText,t.first), EventType:domain.EventTypePlateDetectedV1,
		SchemaVersion:"1", CameraID:t.camera, ObservedAt:t.first, PluginID:"plate-ocr",
		PluginVersion:"1.0.0", Confidence:conf, DedupeKey:dedupe, TrackID:t.id,
		RawText:c.RawText, NormalizedText:c.NormalizedText, Format:c.Format, Candidates:cands,
		DetectorConfidence:t.detectorBest, BBox:t.bbox, Lane:t.lane, Direction:t.direction,
		EvidenceJPEG:append([]byte(nil),t.evidence...),
	}, true
}

func matchScore(t *track, o domain.NormalizedObservation) float64 {
	iou := t.bbox.IoU(o.BBox)
	text := 0.0
	for _, c := range o.Candidates {
		for _, s := range t.scores {
			d := distance(c.NormalizedText,s.candidate.NormalizedText)
			if d==0 { text=1 } else if d==1 && len(c.NormalizedText)==7 && text<.75 { text=.75 }
		}
	}
	return .55*text + .45*iou
}
func trackID(o domain.NormalizedObservation) string {
	h := sha256.Sum256([]byte(o.CameraID+"|"+strconv.FormatInt(o.ObservedAt.UnixMilli(),10)+"|"+
		strconv.FormatFloat(o.BBox.X1,'f',4,64)+"|"+strconv.FormatFloat(o.BBox.Y1,'f',4,64)))
	return hex.EncodeToString(h[:8])
}
func eventID(track, plate string, t time.Time) string {
	h := sha256.Sum256([]byte(track+"|"+plate+"|"+strconv.FormatInt(t.UnixMilli(),10)))
	return hex.EncodeToString(h[:16])
}
func distance(a,b string) int {
	ar,br:=[]rune(a),[]rune(b)
	if len(ar)!=len(br) { if len(ar)>len(br){return len(ar)}; return len(br) }
	d:=0; for i:=range ar { if ar[i]!=br[i] { d++ } }; return d
}
func clamp(v float64) float64 { if v<0{return 0}; if v>1{return 1}; return v }
func maxi(a,b int) int { if a>b{return a}; return b }
func minInt(a,b int) int { if a<b{return a}; return b }
