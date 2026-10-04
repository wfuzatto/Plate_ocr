package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (s *Service) Snapshot()map[string]any{
	s.mu.RLock();lastErr:=s.lastError;s.mu.RUnlock()
	spoolCount,_:=s.spool.Count()
	lastFrame:=s.lastFrameUnixMS.Load()
	lastEvent:=s.lastEventUnixMS.Load()
	state:="healthy"
	if s.cameraCount.Load()==0{state="idle"}
	if s.cameraCount.Load()>0&&lastFrame==0{state="starting"}
	if s.cameraCount.Load()>0&&lastFrame>0&&time.Since(time.UnixMilli(lastFrame))>15*time.Second{state="degraded"}
	return map[string]any{
		"status":state,"version":"1.0.0","provider":"classic-offline",
		"cameras":s.cameraCount.Load(),"frames_fetched":s.framesFetched.Load(),
		"frame_errors":s.frameErrors.Load(),"events_published":s.eventsPublished.Load(),
		"events_spooled":s.eventsSpooled.Load(),"spool_replayed":s.spoolReplayed.Load(),
		"spool_pending":spoolCount,"last_frame_unix_ms":lastFrame,"last_event_unix_ms":lastEvent,
		"last_error":lastErr,
	}
}

func (s *Service) MetricsText()string{
	spoolCount,_:=s.spool.Count()
	var b strings.Builder
	lines:=[][2]string{
		{"plate_ocr_cameras",strconv.FormatInt(s.cameraCount.Load(),10)},
		{"plate_ocr_frames_fetched_total",strconv.FormatUint(s.framesFetched.Load(),10)},
		{"plate_ocr_frame_errors_total",strconv.FormatUint(s.frameErrors.Load(),10)},
		{"plate_ocr_events_published_total",strconv.FormatUint(s.eventsPublished.Load(),10)},
		{"plate_ocr_events_spooled_total",strconv.FormatUint(s.eventsSpooled.Load(),10)},
		{"plate_ocr_spool_replayed_total",strconv.FormatUint(s.spoolReplayed.Load(),10)},
		{"plate_ocr_spool_pending",strconv.Itoa(spoolCount)},
	}
	for _,kv:=range lines{fmt.Fprintf(&b,"%s %s
",kv[0],kv[1])}
	return b.String()
}

func (s *Service) ServeStatus(ctx context.Context)error{
	mux:=http.NewServeMux()
	mux.HandleFunc("GET /healthz",func(w http.ResponseWriter,r *http.Request){
		w.Header().Set("Content-Type","application/json")
		_ = json.NewEncoder(w).Encode(s.Snapshot())
	})
	mux.HandleFunc("GET /metrics",func(w http.ResponseWriter,r *http.Request){
		w.Header().Set("Content-Type","text/plain; version=0.0.4")
		_,_ = w.Write([]byte(s.MetricsText()))
	})
	server:=&http.Server{Addr:s.cfg.ListenAddress,Handler:mux,ReadHeaderTimeout:3*time.Second}
	ln,err:=net.Listen("tcp",s.cfg.ListenAddress)
	if err!=nil{return err}
	errCh:=make(chan error,1)
	go func(){errCh<-server.Serve(ln)}()
	select{
	case<-ctx.Done():
		shutCtx,cancel:=context.WithTimeout(context.Background(),3*time.Second);defer cancel()
		_ = server.Shutdown(shutCtx);return nil
	case err:=<-errCh:
		if err==http.ErrServerClosed{return nil}
		return err
	}
}
