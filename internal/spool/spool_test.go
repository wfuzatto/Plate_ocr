package spool

import (
	"testing"
	"time"
	"github.com/wfuzatto/Plate_ocr/internal/domain"
)

func TestSpoolRoundTrip(t *testing.T){
	s,err:=Open(t.TempDir());if err!=nil{t.Fatal(err)}
	ev:=domain.PlateEvent{EventID:"evt-1",CameraID:"cam",ObservedAt:time.Now(),NormalizedText:"ABC1D23"}
	if err:=s.Enqueue(ev,[]byte{1,2,3});err!=nil{t.Fatal(err)}
	jobs,err:=s.Pending(10);if err!=nil{t.Fatal(err)}
	if len(jobs)!=1||jobs[0].Event.EventID!="evt-1"||len(jobs[0].Evidence)!=3{t.Fatalf("bad jobs: %#v",jobs)}
	if err:=s.Ack(jobs[0]);err!=nil{t.Fatal(err)}
	n,err:=s.Count();if err!=nil||n!=0{t.Fatalf("count=%d err=%v",n,err)}
}
