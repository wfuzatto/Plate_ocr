package classic

import (
	"context"
	"errors"
	"sort"

	"github.com/wfuzatto/Plate_ocr/internal/domain"
	"github.com/wfuzatto/Plate_ocr/internal/inference"
	"github.com/wfuzatto/Plate_ocr/internal/vision"
)

type DetectorConfig struct {
	Threshold  float64
	MaxResults int
	MinWidthPX int
	MinHeightPX int
}

type Detector struct{ cfg DetectorConfig }

func NewDetector(cfg DetectorConfig) *Detector {
	if cfg.Threshold <= 0 { cfg.Threshold = .36 }
	if cfg.MaxResults < 1 { cfg.MaxResults = 8 }
	if cfg.MinWidthPX < 20 { cfg.MinWidthPX = 70 }
	if cfg.MinHeightPX < 8 { cfg.MinHeightPX = 20 }
	return &Detector{cfg:cfg}
}

type scoredDetection struct {
	d inference.Detection
	score float64
}

func (d *Detector) Detect(ctx context.Context, frame inference.Frame) ([]inference.Detection,error) {
	if frame.Image == nil { return nil, errors.New("classic detector requires decoded image") }
	g:=vision.FromImage(frame.Image)
	if g.W<d.cfg.MinWidthPX || g.H<d.cfg.MinHeightPX { return nil,nil }
	dx:=vision.NewIntegral(g,"dx")
	dy:=vision.NewIntegral(g,"dy")
	edges:=vision.NewIntegral(g,"edge")
	var all []scoredDetection
	scales:=[]float64{.11,.16,.22,.30,.40}
	aspects:=[]float64{3.25,1.45}
	for _,scale:=range scales{
		select { case <-ctx.Done(): return nil,ctx.Err(); default: }
		w:=int(float64(g.W)*scale)
		if w<d.cfg.MinWidthPX { w=d.cfg.MinWidthPX }
		if w>g.W { continue }
		for _,aspect:=range aspects{
			h:=int(float64(w)/aspect)
			if h<d.cfg.MinHeightPX { h=d.cfg.MinHeightPX }
			if h>=g.H { continue }
			stepX:=maxInt(6,w/5)
			stepY:=maxInt(4,h/3)
			for y:=0;y+h<=g.H;y+=stepY{
				for x:=0;x+w<=g.W;x+=stepX{
					box:=domain.BBox{
						X1:float64(x)/float64(g.W),Y1:float64(y)/float64(g.H),
						X2:float64(x+w)/float64(g.W),Y2:float64(y+h)/float64(g.H),
					}
					if frame.ROI!=nil && !box.CenterInside(*frame.ROI) { continue }
					area:=float64(w*h*255)
					v:=float64(dx.Rect(x,y,x+w,y+h))/area
					hg:=float64(dy.Rect(x,y,x+w,y+h))/area
					ed:=float64(edges.Rect(x,y,x+w,y+h))/area
					score:=clamp01(v*2.9 + hg*1.1 + ed*1.8)
					if score<d.cfg.Threshold { continue }
					all=append(all,scoredDetection{d:inference.Detection{
						BBox:box,Confidence:score,Lane:frame.Lane,Direction:frame.Direction,
					},score:score})
				}
			}
		}
	}
	sort.Slice(all,func(i,j int)bool{return all[i].score>all[j].score})
	out:=make([]inference.Detection,0,d.cfg.MaxResults)
	for _,cand:=range all{
		keep:=true
		for _,accepted:=range out{
			if cand.d.BBox.IoU(accepted.BBox)>.42 { keep=false; break }
		}
		if keep {
			out=append(out,cand.d)
			if len(out)>=d.cfg.MaxResults { break }
		}
	}
	return out,nil
}

func clamp01(v float64)float64{if v<0{return 0};if v>1{return 1};return v}
func maxInt(a,b int)int{if a>b{return a};return b}
