package pipeline

import (
	"context"
	"image"
	"testing"
	"time"

	"github.com/wfuzatto/Plate_ocr/internal/app"
	"github.com/wfuzatto/Plate_ocr/internal/config"
	"github.com/wfuzatto/Plate_ocr/internal/domain"
	"github.com/wfuzatto/Plate_ocr/internal/inference"
)

type fakeDetector struct{}
func (fakeDetector) Detect(context.Context,inference.Frame)([]inference.Detection,error) {
	return []inference.Detection{{BBox:domain.BBox{X1:.1,Y1:.2,X2:.4,Y2:.5},Confidence:.96}},nil
}
type fakeRecognizer struct{}
func (fakeRecognizer) Recognize(context.Context,inference.Frame,inference.Detection)([]domain.RawOCRCandidate,error) {
	return []domain.RawOCRCandidate{{Text:"ABC1D23",Confidence:.94}},nil
}

func TestPipeline(t *testing.T) {
	cfg:=config.Default()
	cfg.MinEventConfidence=.5
	p,err:=New(fakeDetector{},fakeRecognizer{},app.New(cfg))
	if err!=nil { t.Fatal(err) }
	now:=time.Unix(100,0)
	for i:=0;i<2;i++ {
		_,err=p.ProcessFrame(context.Background(),inference.Frame{
			CameraID:"cam",ObservedAt:now.Add(time.Duration(i)*100*time.Millisecond),
			Width:1920,Height:1080,Format:inference.PixelNV12,Image:image.NewGray(image.Rect(0,0,1920,1080)),
		})
		if err!=nil { t.Fatal(err) }
	}
	got:=p.FlushAll()
	if len(got)!=1 || got[0].NormalizedText!="ABC1D23" { t.Fatalf("unexpected: %#v",got) }
}
