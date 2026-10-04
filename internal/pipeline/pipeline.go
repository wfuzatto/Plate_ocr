package pipeline

import (
	"context"
	"errors"

	"github.com/wfuzatto/Plate_ocr/internal/app"
	"github.com/wfuzatto/Plate_ocr/internal/domain"
	"github.com/wfuzatto/Plate_ocr/internal/inference"
)

type Pipeline struct {
	detector inference.Detector
	recognizer inference.Recognizer
	engine *app.Engine
}

func New(detector inference.Detector, recognizer inference.Recognizer, engine *app.Engine) (*Pipeline,error) {
	if detector==nil || recognizer==nil || engine==nil {
		return nil,errors.New("detector, recognizer and engine are required")
	}
	return &Pipeline{detector:detector,recognizer:recognizer,engine:engine},nil
}

func (p *Pipeline) ProcessFrame(ctx context.Context, frame inference.Frame) ([]domain.PlateEvent,error) {
	if frame.CameraID=="" || frame.Width<=0 || frame.Height<=0 || len(frame.Data)==0 {
		return nil,errors.New("invalid frame")
	}
	detections,err:=p.detector.Detect(ctx,frame)
	if err!=nil { return nil,err }
	var events []domain.PlateEvent
	for _,detection:=range detections {
		if !detection.BBox.Valid() || detection.Confidence<=0 { continue }
		ocr,err:=p.recognizer.Recognize(ctx,frame,detection)
		if err!=nil { return events,err }
		got,err:=p.engine.Process(domain.Observation{
			CameraID:frame.CameraID, ObservedAt:frame.ObservedAt,
			DetectorConfidence:detection.Confidence, BBox:detection.BBox, OCR:ocr,
			Lane:detection.Lane, Direction:detection.Direction,
		})
		if err!=nil { return events,err }
		events=append(events,got...)
	}
	return events,nil
}
func (p *Pipeline) FlushAll() []domain.PlateEvent { return p.engine.FlushAll() }
