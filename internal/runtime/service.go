package runtime

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wfuzatto/Plate_ocr/internal/config"
	"github.com/wfuzatto/Plate_ocr/internal/domain"
	"github.com/wfuzatto/Plate_ocr/internal/inference"
	"github.com/wfuzatto/Plate_ocr/internal/pipeline"
	"github.com/wfuzatto/Plate_ocr/internal/runtime/nvrclient"
	"github.com/wfuzatto/Plate_ocr/internal/spool"
)

type Service struct {
	cfg config.Config
	client *nvrclient.Client
	pipeline *pipeline.Pipeline
	spool *spool.Store

	mu sync.RWMutex
	cameras []nvrclient.Camera
	lastError string

	framesFetched atomic.Uint64
	frameErrors atomic.Uint64
	eventsPublished atomic.Uint64
	eventsSpooled atomic.Uint64
	spoolReplayed atomic.Uint64
	lastFrameUnixMS atomic.Int64
	lastEventUnixMS atomic.Int64
	cameraCount atomic.Int64
}

func NewService(cfg config.Config,client *nvrclient.Client,p *pipeline.Pipeline,sp *spool.Store)(*Service,error){
	if client==nil||p==nil||sp==nil{return nil,fmt.Errorf("client, pipeline and spool are required")}
	return &Service{cfg:cfg,client:client,pipeline:p,spool:sp},nil
}

func (s *Service) Run(ctx context.Context) error {
	defer func(){
		for _,ev:=range s.pipeline.FlushAll(){
			if err:=s.send(ctx,ev,ev.EvidenceJPEG);err!=nil{_ = s.spool.Enqueue(ev,ev.EvidenceJPEG)}
		}
	}()
	_ = s.refreshCameras(ctx)
	_ = s.flushSpool(ctx)
	s.pollOnce(ctx)

	frameEvery:=time.Duration(float64(time.Second)/s.cfg.PollFPS)
	if frameEvery<50*time.Millisecond{frameEvery=50*time.Millisecond}
	frameTicker:=time.NewTicker(frameEvery);defer frameTicker.Stop()
	cameraTicker:=time.NewTicker(15*time.Second);defer cameraTicker.Stop()
	spoolTicker:=time.NewTicker(3*time.Second);defer spoolTicker.Stop()

	for{
		select{
		case<-ctx.Done():return nil
		case<-cameraTicker.C:
			if err:=s.refreshCameras(ctx);err!=nil{s.setError(err)}
		case<-spoolTicker.C:
			if err:=s.flushSpool(ctx);err!=nil{s.setError(err)}
		case<-frameTicker.C:
			s.pollOnce(ctx)
		}
	}
}

func (s *Service) refreshCameras(ctx context.Context)error{
	items,err:=s.client.ListCameras(ctx)
	if err!=nil{return err}
	selected:=make([]nvrclient.Camera,0,len(items))
	allow:=make(map[string]bool,len(s.cfg.CameraIDs))
	for _,id:=range s.cfg.CameraIDs{allow[id]=true}
	for _,cam:=range items{
		if !cam.Enabled||!cam.SnapshotAvailable{continue}
		if len(allow)>0&&!allow[cam.ID]{continue}
		cc,ok:=s.cfg.Cameras[cam.ID]
		if ok&&cc.Enabled!=nil&&!*cc.Enabled{continue}
		selected=append(selected,cam)
	}
	s.mu.Lock();s.cameras=selected;s.mu.Unlock()
	s.cameraCount.Store(int64(len(selected)))
	return nil
}

func (s *Service) pollOnce(ctx context.Context){
	s.mu.RLock()
	cams:=append([]nvrclient.Camera(nil),s.cameras...)
	s.mu.RUnlock()
	if len(cams)==0{return}
	sem:=make(chan struct{},s.cfg.MaxParallel)
	var wg sync.WaitGroup
	for _,cam:=range cams{
		cam:=cam
		wg.Add(1)
		go func(){
			defer wg.Done()
			select{case sem<-struct{}{}:case<-ctx.Done():return}
			defer func(){<-sem}()
			if err:=s.processCamera(ctx,cam);err!=nil{s.frameErrors.Add(1);s.setError(err)}
		}()
	}
	wg.Wait()
}

func (s *Service) processCamera(ctx context.Context,cam nvrclient.Camera)error{
	jpeg,observed,err:=s.client.FetchFrame(ctx,cam.ID)
	if err!=nil{return fmt.Errorf("camera %s frame: %w",cam.ID,err)}
	s.framesFetched.Add(1)
	s.lastFrameUnixMS.Store(observed.UnixMilli())
	cc:=s.cfg.Cameras[cam.ID]
	frame:=inference.Frame{
		FrameID:fmt.Sprintf("%s-%d",cam.ID,observed.UnixNano()),
		CameraID:cam.ID,ObservedAt:observed,Format:inference.PixelJPEG,Data:jpeg,
		ROI:cc.ROI,Lane:cc.Lane,Direction:cc.Direction,
	}
	events,err:=s.pipeline.ProcessFrame(ctx,frame)
	if err!=nil{return fmt.Errorf("camera %s pipeline: %w",cam.ID,err)}
	for _,ev:=range events{
		if err:=s.send(ctx,ev,ev.EvidenceJPEG);err!=nil{
			s.eventsSpooled.Add(1)
			if qerr:=s.spool.Enqueue(ev,ev.EvidenceJPEG);qerr!=nil{return fmt.Errorf("publish: %v; spool: %w",err,qerr)}
			s.setError(err)
		}
	}
	return nil
}

func (s *Service) send(ctx context.Context,ev domain.PlateEvent,evidence []byte)error{
	if s.cfg.EvidenceEnabled&&ev.SnapshotRef==""&&len(evidence)>0{
		ref,err:=s.client.UploadEvidence(ctx,ev.EventID,evidence)
		if err!=nil{return err}
		ev.SnapshotRef=ref
	}
	ev.EvidenceJPEG=nil
	if err:=s.client.PublishEvent(ctx,ev);err!=nil{return err}
	s.eventsPublished.Add(1)
	s.lastEventUnixMS.Store(time.Now().UTC().UnixMilli())
	return nil
}

func (s *Service) flushSpool(ctx context.Context)error{
	jobs,err:=s.spool.Pending(50)
	if err!=nil{return err}
	for _,job:=range jobs{
		if err:=s.send(ctx,job.Event,job.Evidence);err!=nil{return err}
		if err:=s.spool.Ack(job);err!=nil{return err}
		s.spoolReplayed.Add(1)
	}
	return nil
}

func (s *Service) setError(err error){
	if err==nil{return}
	s.mu.Lock();s.lastError=err.Error();s.mu.Unlock()
}
