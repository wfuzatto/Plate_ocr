package classic

import (
	"context"
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/wfuzatto/Plate_ocr/internal/domain"
	"github.com/wfuzatto/Plate_ocr/internal/inference"
)

func TestRecognizerSyntheticMercosul(t *testing.T){
	img:=renderPlate("ABC1D23",350,90)
	r:=NewRecognizer(RecognizerConfig{Threshold:.35})
	out,err:=r.Recognize(context.Background(),inference.Frame{
		CameraID:"cam",ObservedAt:time.Now(),Image:img,Width:350,Height:90,
	},inference.Detection{BBox:domain.BBox{X1:0,Y1:0,X2:1,Y2:1},Confidence:.9})
	if err!=nil{t.Fatal(err)}
	if len(out)==0{t.Fatal("expected OCR result")}
	found:=false
	for _,v:=range out{if v.Text=="ABC1D23"{found=true}}
	if !found{t.Fatalf("expected ABC1D23, got %#v",out)}
}

func TestDetectorFindsSyntheticPlate(t *testing.T){
	img:=image.NewGray(image.Rect(0,0,800,450))
	for i:=range img.Pix{img.Pix[i]=80}
	plate:=renderPlate("ABC1D23",320,82)
	for y:=0;y<82;y++{for x:=0;x<320;x++{img.SetGray(220+x,180+y,plate.GrayAt(x,y))}}
	d:=NewDetector(DetectorConfig{Threshold:.18,MaxResults:8,MinWidthPX:70,MinHeightPX:20})
	out,err:=d.Detect(context.Background(),inference.Frame{Image:img,Width:800,Height:450})
	if err!=nil{t.Fatal(err)}
	if len(out)==0{t.Fatal("expected at least one detection")}
	target:=domain.BBox{X1:220.0/800,Y1:180.0/450,X2:540.0/800,Y2:262.0/450}
	best:=0.0
	for _,v:=range out{if i:=v.BBox.IoU(target);i>best{best=i}}
	if best<.15{t.Fatalf("expected detection near plate, best IoU=%f",best)}
}

func renderPlate(text string,w,h int)*image.Gray{
	img:=image.NewGray(image.Rect(0,0,w,h))
	for i:=range img.Pix{img.Pix[i]=245}
	marginX:=int(float64(w)*.06)
	marginY:=int(float64(h)*.16)
	cellW:=(w-2*marginX)/7
	for i,ch:=range []rune(text){
		p:=glyphs[ch]
		xBase:=marginX+i*cellW
		for gy:=0;gy<7;gy++{
			for gx:=0;gx<5;gx++{
				if p[gy][gx]!='1'{continue}
				x0:=xBase+gx*cellW/5
				x1:=xBase+(gx+1)*cellW/5
				y0:=marginY+gy*(h-2*marginY)/7
				y1:=marginY+(gy+1)*(h-2*marginY)/7
				for y:=y0;y<y1;y++{for x:=x0;x<x1;x++{img.SetGray(x,y,color.Gray{Y:10})}}
			}
		}
	}
	return img
}
