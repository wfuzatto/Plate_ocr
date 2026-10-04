package nvrclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/wfuzatto/Plate_ocr/internal/domain"
)

func TestClientProtocol(t *testing.T){
	var gotEvent bool
	srv:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		if r.Header.Get("Authorization")!="Bearer secret"{t.Errorf("missing auth")}
		switch r.URL.Path{
		case "/api/v1/plugin/v1/cameras":
			_ = json.NewEncoder(w).Encode(map[string]any{"items":[]map[string]any{{"id":"cam-1","enabled":true,"snapshot_available":true}}})
		case "/api/v1/plugin/v1/cameras/cam-1/frame":
			w.Header().Set("Content-Type","image/jpeg");w.Header().Set("X-NVR-Observed-At","2026-10-04T20:00:00Z");_,_=w.Write([]byte{0xff,0xd8,0xff,0xd9})
		case "/api/v1/plugin/v1/evidence":
			w.WriteHeader(http.StatusCreated);_ = json.NewEncoder(w).Encode(map[string]string{"snapshot_ref":"evidence/evt.jpg"})
		case "/api/v1/plugin/v1/events":
			gotEvent=true;w.WriteHeader(http.StatusCreated);_,_=w.Write([]byte("{}"))
		default:http.NotFound(w,r)
		}
	}));defer srv.Close()
	c,err:=New(srv.URL,"secret",time.Second);if err!=nil{t.Fatal(err)}
	cams,err:=c.ListCameras(context.Background());if err!=nil||len(cams)!=1{t.Fatalf("cams=%v err=%v",cams,err)}
	_,at,err:=c.FetchFrame(context.Background(),"cam-1");if err!=nil||at.IsZero(){t.Fatalf("frame err=%v at=%v",err,at)}
	ref,err:=c.UploadEvidence(context.Background(),"evt",[]byte{1});if err!=nil||ref==""{t.Fatalf("evidence %q err=%v",ref,err)}
	err=c.PublishEvent(context.Background(),domain.PlateEvent{EventID:"evt",EventType:domain.EventTypePlateDetectedV1,SchemaVersion:"1",CameraID:"cam-1",ObservedAt:time.Now(),PluginID:"plate-ocr",PluginVersion:"1.0.0",NormalizedText:"ABC1D23"})
	if err!=nil||!gotEvent{t.Fatalf("publish err=%v got=%v",err,gotEvent)}
}
