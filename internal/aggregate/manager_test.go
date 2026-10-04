package aggregate

import (
	"testing"
	"time"

	"github.com/wfuzatto/Plate_ocr/internal/domain"
)

func TestAggregateTwoFrames(t *testing.T) {
	m:=New(Config{TrackTTL:time.Second,MinObservations:2,MinConfidence:.5,MaxActivePerCamera:16})
	now:=time.Unix(100,0)
	for i:=0;i<2;i++ {
		m.Observe(domain.NormalizedObservation{
			CameraID:"cam-1", ObservedAt:now.Add(time.Duration(i)*100*time.Millisecond),
			DetectorConfidence:.95, BBox:domain.BBox{X1:.1,Y1:.2,X2:.4,Y2:.5},
			Candidates:[]domain.Candidate{{RawText:"ABC1D23",NormalizedText:"ABC1D23",Format:"BR_MERCOSUL",Confidence:.92}},
		})
	}
	e:=m.FlushExpired(now.Add(2*time.Second))
	if len(e)!=1 || e[0].NormalizedText!="ABC1D23" { t.Fatalf("unexpected events: %#v",e) }
}
