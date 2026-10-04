package vision

type Integral struct {
	W,H int
	Sum []uint64
}

func NewIntegral(g Gray, mode string) Integral {
	w,h:=g.W,g.H
	in:=Integral{W:w+1,H:h+1,Sum:make([]uint64,(w+1)*(h+1))}
	for y:=0;y<h;y++{
		var row uint64
		for x:=0;x<w;x++{
			var v uint8
			switch mode {
			case "dx":
				a,b:=int(g.At(x-1,y)),int(g.At(x+1,y)); if a>b { v=uint8(a-b) } else { v=uint8(b-a) }
			case "dy":
				a,b:=int(g.At(x,y-1)),int(g.At(x,y+1)); if a>b { v=uint8(a-b) } else { v=uint8(b-a) }
			case "edge":
				dx:=abs(int(g.At(x-1,y))-int(g.At(x+1,y)))
				dy:=abs(int(g.At(x,y-1))-int(g.At(x,y+1)))
				if dx+dy>70 { v=255 }
			default:
				v=g.At(x,y)
			}
			row += uint64(v)
			in.Sum[(y+1)*in.W+(x+1)] = in.Sum[y*in.W+(x+1)] + row
		}
	}
	return in
}

func (in Integral) Rect(x0,y0,x1,y1 int) uint64 {
	if x0<0{x0=0}; if y0<0{y0=0}; if x1>in.W-1{x1=in.W-1}; if y1>in.H-1{y1=in.H-1}
	if x1<=x0||y1<=y0{return 0}
	a:=in.Sum[y0*in.W+x0]
	b:=in.Sum[y0*in.W+x1]
	c:=in.Sum[y1*in.W+x0]
	d:=in.Sum[y1*in.W+x1]
	return d-b-c+a
}
func abs(v int) int { if v<0{return -v}; return v }
