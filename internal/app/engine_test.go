package app

import (
	"testing"
	"time"

	"github.com/wfuzatto/Plate_ocr/internal/config"
	"github.com/wfuzatto/Plate_ocr/internal/domain"
)

func TestEndToEndCore(t *testing.T) {
	cfg:=config.Default()
	cfg.MinEventConfidence=.5
	e:=New(cfg)
	n:=time.Unix(100,0)
	for i:=0;i<2;i++ {
		_,err:=e.Process(domain.Observation{
			CameraID:"cam", ObservedAt:n.Add(time.Duration(i)*100*time.Millisecond),
			DetectorConfidence:.96, BBox:domain.BBox{X1:.1,Y1:.2,X2:.4,Y2:.5},
			OCR:[]domain.RawOCRCandidate{{Text:"ABC1D23",Confidence:.94}},
		})
		if err!=nil { t.Fatal(err) }
	}
	got:=e.FlushAll()
	if len(got)!=1 || got[0].Format!="BR_MERCOSUL" { t.Fatalf("unexpected: %#v",got) }
}
