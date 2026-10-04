package classic

import (
	"context"
	"errors"
	"sort"

	"github.com/wfuzatto/Plate_ocr/internal/domain"
	"github.com/wfuzatto/Plate_ocr/internal/inference"
	"github.com/wfuzatto/Plate_ocr/internal/vision"
)

type RecognizerConfig struct {
	Threshold float64
}

type Recognizer struct{ cfg RecognizerConfig }

func NewRecognizer(cfg RecognizerConfig)*Recognizer{
	if cfg.Threshold<=0{cfg.Threshold=.48}
	return &Recognizer{cfg:cfg}
}

type recognized struct{
	text string
	conf float64
}

func (r *Recognizer) Recognize(ctx context.Context, frame inference.Frame, det inference.Detection)([]domain.RawOCRCandidate,error){
	if frame.Image==nil{return nil,errors.New("classic recognizer requires decoded image")}
	b:=frame.Image.Bounds()
	x0:=int(det.BBox.X1*float64(b.Dx()))
	y0:=int(det.BBox.Y1*float64(b.Dy()))
	x1:=int(det.BBox.X2*float64(b.Dx()))
	y1:=int(det.BBox.Y2*float64(b.Dy()))
	if x1-x0<30||y1-y0<10{return nil,nil}
	g:=vision.FromImage(frame.Image).Crop(x0,y0,x1,y1)
	if g.W==0||g.H==0{return nil,nil}
	select{case<-ctx.Done():return nil,ctx.Err();default:}

	best:=map[string]recognized{}
	layouts:=[]bool{false}
	if float64(g.W)/float64(g.H)<2.05{layouts=[]bool{true,false}}
	yMargins:=[]float64{.10,.16,.22}
	xMargins:=[]float64{.02,.06,.10}
	for _,twoLine:=range layouts{
		for _,ym:=range yMargins{
			for _,xm:=range xMargins{
				for _,invert:=range []bool{false,true}{
					for _,pat:=range patterns{
						rec:=recognizePattern(g,pat,twoLine,xm,ym,invert)
						if rec.conf<r.cfg.Threshold{continue}
						if old,ok:=best[rec.text];!ok||rec.conf>old.conf{best[rec.text]=rec}
					}
				}
			}
		}
	}
	arr:=make([]recognized,0,len(best))
	for _,v:=range best{arr=append(arr,v)}
	sort.Slice(arr,func(i,j int)bool{return arr[i].conf>arr[j].conf})
	if len(arr)>4{arr=arr[:4]}
	out:=make([]domain.RawOCRCandidate,0,len(arr))
	for _,v:=range arr{out=append(out,domain.RawOCRCandidate{Text:v.text,Confidence:clamp01(v.conf)})}
	return out,nil
}

func recognizePattern(g vision.Gray,pat platePattern,twoLine bool,xMargin,yMargin float64,invert bool)recognized{
	var text []rune
	total:=0.0
	for i,class:=range pat.classes{
		var x0,y0,x1,y1 int
		if twoLine{
			if i<3{
				usableW:=float64(g.W)*(1-2*xMargin)
				cell:=usableW/3
				x0=int(float64(g.W)*xMargin+float64(i)*cell)
				x1=int(float64(g.W)*xMargin+float64(i+1)*cell)
				y0=int(float64(g.H)*yMargin)
				y1=int(float64(g.H)*.48)
			}else{
				j:=i-3
				usableW:=float64(g.W)*(1-2*xMargin)
				cell:=usableW/4
				x0=int(float64(g.W)*xMargin+float64(j)*cell)
				x1=int(float64(g.W)*xMargin+float64(j+1)*cell)
				y0=int(float64(g.H)*.52)
				y1=int(float64(g.H)*(1-yMargin))
			}
		}else{
			usableW:=float64(g.W)*(1-2*xMargin)
			cell:=usableW/7
			x0=int(float64(g.W)*xMargin+float64(i)*cell)
			x1=int(float64(g.W)*xMargin+float64(i+1)*cell)
			y0=int(float64(g.H)*yMargin)
			y1=int(float64(g.H)*(1-yMargin))
		}
		cell:=g.Crop(x0,y0,x1,y1)
		ch,score:=matchCell(cell,class,invert)
		text=append(text,ch)
		total+=score
	}
	return recognized{text:string(text),conf:total/7}
}

func matchCell(g vision.Gray,class byte,invert bool)(rune,float64){
	if g.W<2||g.H<2{return '?',0}
	threshold:=otsu(g)
	var allowed []rune
	if class=='L'{allowed=letters}else{allowed=digits}
	bestRune:='?'
	best:=-1.0
	for _,ch:=range allowed{
		tmpl:=glyphs[ch]
		diff:=0
		for gy:=0;gy<7;gy++{
			for gx:=0;gx<5;gx++{
				x0:=gx*g.W/5
				x1:=(gx+1)*g.W/5
				y0:=gy*g.H/7
				y1:=(gy+1)*g.H/7
				cx:=(x0+x1-1)/2
				cy:=(y0+y1-1)/2
				center:=g.At(cx,cy)
				fg:=center<=threshold
				if invert{fg=!fg}
				want:=tmpl[gy][gx]=='1'
				if fg!=want{diff++}
			}
		}
		score:=1-float64(diff)/35
		if score>best{best=score;bestRune=ch}
	}
	return bestRune,best
}

func otsu(g vision.Gray)uint8{
	var hist [256]uint64
	for _,v:=range g.Pix{hist[v]++}
	total:=uint64(len(g.Pix))
	if total==0{return 127}
	var sum uint64
	for i,h:=range hist{sum+=uint64(i)*h}
	var sumB,wB uint64
	bestVar:=-1.0
	best:=127
	for t:=0;t<256;t++{
		wB+=hist[t]
		if wB==0{continue}
		wF:=total-wB
		if wF==0{break}
		sumB+=uint64(t)*hist[t]
		mB:=float64(sumB)/float64(wB)
		mF:=float64(sum-sumB)/float64(wF)
		d:=mB-mF
		v:=float64(wB)*float64(wF)*d*d
		if v>bestVar{bestVar=v;best=t}
	}
	return uint8(best)
}
