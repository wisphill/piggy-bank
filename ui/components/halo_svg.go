package components

import (
	"image"
	"image/color"
	"piggy-bank/assets"
	"piggy-bank/ui/theme"
	"sync"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
)

// discRatio is the disc radius relative to the emblem size, matching the
// ring geometry in assets/status_halo.svg.
const discRatio = 0.2693

type haloKey struct {
	size int
	c    color.NRGBA
}

var (
	haloMu    sync.Mutex
	haloCache = map[haloKey]*SVGRenderer{}
)

func haloRing(sz int, c color.NRGBA) *SVGRenderer {
	key := haloKey{sz, c}
	haloMu.Lock()
	defer haloMu.Unlock()
	r, ok := haloCache[key]
	if !ok {
		var err error
		if r, err = LoadSVG(assets.StatusHaloSVG, sz, sz, c); err != nil {
			return nil
		}
		haloCache[key] = r
	}
	return r
}

// StatusHaloSVG draws the status emblem: a solid disc inside the ring of
// eight identical segments from assets/status_halo.svg, tinted with c. The
// ring is rotated by rotation radians. center is drawn on the disc in white.
func StatusHaloSVG(gtx layout.Context, size unit.Dp, c color.NRGBA, rotation float32, center *widget.Icon) layout.Dimensions {
	sz := gtx.Dp(size)
	mid := float32(sz) / 2

	discR := float32(sz) * discRatio
	paint.FillShape(gtx.Ops, c, clip.Ellipse{
		Min: image.Pt(int(mid-discR), int(mid-discR)),
		Max: image.Pt(int(mid+discR+0.5), int(mid+discR+0.5)),
	}.Op(gtx.Ops))

	if r := haloRing(sz, c); r != nil {
		rot := op.Affine(f32.Affine2D{}.Rotate(f32.Pt(mid, mid), rotation)).Push(gtx.Ops)
		r.Layout(gtx)
		rot.Pop()
	}

	if center != nil {
		iconSz := int(float32(sz) * 0.26)
		off := (sz - iconSz) / 2
		stack := op.Offset(image.Pt(off, off)).Push(gtx.Ops)
		cgtx := gtx
		cgtx.Constraints = layout.Exact(image.Pt(iconSz, iconSz))
		center.Layout(cgtx, theme.OnPrimary)
		stack.Pop()
	}
	return layout.Dimensions{Size: image.Pt(sz, sz)}
}
