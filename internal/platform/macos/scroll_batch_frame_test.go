//go:build darwin && !ios

package macos

import (
	"math"
	"testing"

	"github.com/ebitengine/purego/objc"
	"github.com/nsgo-dev/nsgo/internal/driver"
	"github.com/nsgo-dev/nsgo/internal/platform/apple"
)

func scrollBatchLayerBounds(layer, root objc.ID) driver.Bounds {
	b := objc.Send[apple.CGRect](layer, selBounds)
	r := objc.Send[apple.CGRect](root, selBounds)
	var quad [4]driver.Point
	for i, p := range [4]apple.CGPoint{b.Origin, {X: b.Origin.X + b.Size.Width, Y: b.Origin.Y},
		{X: b.Origin.X + b.Size.Width, Y: b.Origin.Y + b.Size.Height}, {X: b.Origin.X, Y: b.Origin.Y + b.Size.Height}} {
		at := objc.Send[apple.CGPoint](layer, sel("convertPoint:toLayer:"), p, root)
		quad[i] = driver.Point{X: driver.Points(at.X - r.Origin.X), Y: driver.Points(at.Y - r.Origin.Y)}
	}
	bounds, _ := driver.PointBounds(quad)
	return bounds
}

type scrollBatchFrameCheckpoint struct {
	driver.NativeComponent
	check func()
}

func (c *scrollBatchFrameCheckpoint) Command(string, any) { c.check() }
func (c *scrollBatchFrameCheckpoint) Close()              { c.check = nil }

func TestScrollBatchPlacesNewRowsBeforeDisplay(t *testing.T) {
	onMain(func() {
		loadAppKit()
		const width, height, bottomPad = 562, 800, 190
		window := objc.ID(apple.Class("NSWindow")).Send(selAlloc).Send(sel("initWithContentRect:styleMask:backing:defer:"),
			apple.Rect(40, 40, width, height), uint(0), uint(2), false)
		window.Send(sel("setReleasedWhenClosed:"), false)
		s := newSurface(Options{Width: width, Height: height}, false)
		s.window = window
		window.Send(sel("setContentView:"), s.native)
		defer func() {
			s.close()
			window.Send(sel("close"))
			window.Send(selRelease)
			delete(views, s.native)
			s.native.Send(selRelease)
		}()
		style := driver.DefaultViewStyle
		config := driver.ScrollConfig{ZoomScale: 1}
		old := &driver.MountBatch{Root: 1, Items: []driver.MountItem{
			{Kind: driver.ItemCreate, ID: 1, View: driver.KindView},
			{Kind: driver.ItemFrame, ID: 1, Frame: driver.Bounds{Size: driver.Size{Width: width, Height: height}}},
			{Kind: driver.ItemViewStyle, ID: 1, ViewStyle: &style},
			{Kind: driver.ItemCreate, ID: 2, View: driver.KindScroll},
			{Kind: driver.ItemInsert, ID: 2, Parent: 1},
			{Kind: driver.ItemFrame, ID: 2, Frame: driver.Bounds{Size: driver.Size{Width: width, Height: height}}},
			{Kind: driver.ItemViewStyle, ID: 2, ViewStyle: &style},
			{Kind: driver.ItemScrollConfig, ID: 2, Scroll: &config},
			{Kind: driver.ItemCreate, ID: 3, View: driver.KindView},
			{Kind: driver.ItemInsert, ID: 3, Parent: 2},
			{Kind: driver.ItemFrame, ID: 3, Frame: driver.Bounds{Origin: driver.Point{Y: 1000 - bottomPad - 24}, Size: driver.Size{Width: width, Height: 24}}},
			{Kind: driver.ItemViewStyle, ID: 3, ViewStyle: &style},
			{Kind: driver.ItemScrollContent, ID: 2, Size: driver.Size{Width: width, Height: 1000}},
			{Kind: driver.ItemScrollTo, ID: 2, Point: driver.Point{Y: 1000 - height}},
		}}
		s.apply(old)
		window.Send(sel("orderFront:"), objc.ID(0))
		window.Send(sel("displayIfNeeded"))
		objc.ID(apple.Class("CATransaction")).Send(sel("flush"))
		until := objc.ID(apple.Class("NSDate")).Send(sel("dateWithTimeIntervalSinceNow:"), 0.03)
		objc.ID(apple.Class("NSRunLoop")).Send(sel("mainRunLoop")).Send(sel("runUntilDate:"), until)
		before := scrollBatchLayerBounds(s.views[3].native.Send(selLayer), s.native.Send(selLayer))
		if math.Abs(float64(before.Origin.Y-(height-bottomPad-24))) > 0.5 {
			t.Errorf("settled source layer y=%.1f; want %d", before.Origin.Y, height-bottomPad-24)
		}

		const content = 3200
		items := []driver.MountItem{{Kind: driver.ItemDelete, ID: 3}}
		want := make(map[driver.NodeID]driver.Points)
		for i := range 8 {
			id := driver.NodeID(10 + i)
			y := driver.Points(content - bottomPad - 24 - (7-i)*60)
			want[id] = y - (content - height)
			items = append(items,
				driver.MountItem{Kind: driver.ItemCreate, ID: id, View: driver.KindView},
				driver.MountItem{Kind: driver.ItemInsert, ID: id, Parent: 2},
				driver.MountItem{Kind: driver.ItemFrame, ID: id, Frame: driver.Bounds{Origin: driver.Point{Y: y}, Size: driver.Size{Width: width, Height: 24}}},
				driver.MountItem{Kind: driver.ItemViewStyle, ID: id, ViewStyle: &style})
		}
		items = append(items,
			driver.MountItem{Kind: driver.ItemScrollContent, ID: 2, Size: driver.Size{Width: width, Height: content}},
			driver.MountItem{Kind: driver.ItemScrollTo, ID: 2, Point: driver.Point{Y: content - height}})
		s.apply(&driver.MountBatch{Root: 1, Items: items})
		for id, y := range want {
			got := scrollBatchLayerBounds(s.views[id].native.Send(selLayer), s.native.Send(selLayer))
			if math.Abs(float64(got.Origin.Y-y)) > 0.5 || got.Size.Height != 24 {
				t.Errorf("first applied row %d: native layer y=%.1f height=%.1f; want y=%.1f height=24 before the next display", id, got.Origin.Y, got.Size.Height, y)
			}
		}
		window.Send(sel("displayIfNeeded"))
		objc.ID(apple.Class("CATransaction")).Send(sel("flush"))
		until = objc.ID(apple.Class("NSDate")).Send(sel("dateWithTimeIntervalSinceNow:"), 0.03)
		objc.ID(apple.Class("NSRunLoop")).Send(sel("mainRunLoop")).Send(sel("runUntilDate:"), until)
		for id, y := range want {
			got := scrollBatchLayerBounds(s.views[id].native.Send(selLayer), s.native.Send(selLayer))
			if math.Abs(float64(got.Origin.Y-y)) > 0.5 {
				t.Errorf("settled target row %d: layer y=%.1f; want y=%.1f", id, got.Origin.Y, y)
			}
		}
		const delta = 300
		checkpoint := &scrollBatchFrameCheckpoint{check: func() {
			objc.ID(apple.Class("CATransaction")).Send(sel("flush"))
			window.Send(sel("displayIfNeeded"))
			at := objc.ID(apple.Class("NSDate")).Send(sel("dateWithTimeIntervalSinceNow:"), 0.04)
			objc.ID(apple.Class("NSRunLoop")).Send(sel("mainRunLoop")).Send(sel("runUntilDate:"), at)
			layer := s.views[10].native.Send(selLayer)
			if layer.Send(sel("presentationLayer")) == 0 {
				t.Error("the native test window has no presented row")
				return
			}
			got, ok := apple.LayerBounds(layer, s.native.Send(selLayer))
			if !ok || math.Abs(float64(got.Origin.Y-want[10])) > 0.5 {
				t.Errorf("anchoring exposed half a native batch: presented row y=%.1f; want y=%.1f", got.Origin.Y, want[10])
			}
		}}
		s.views[10].component = checkpoint
		anchored := []driver.MountItem{
			{Kind: driver.ItemScrollContent, ID: 2, Size: driver.Size{Width: width, Height: content + delta}},
			{Kind: driver.ItemScrollBy, ID: 2, Point: driver.Point{Y: delta}},
			{Kind: driver.ItemNativeCommand, ID: 10, Value: "frame-checkpoint"},
		}
		for id := range want {
			frame := s.views[id].frame
			frame.Origin.Y += delta
			anchored = append(anchored, driver.MountItem{Kind: driver.ItemFrame, ID: id, Frame: frame})
		}
		s.apply(&driver.MountBatch{Root: 1, Items: anchored})
		for id, y := range want {
			got := scrollBatchLayerBounds(s.views[id].native.Send(selLayer), s.native.Send(selLayer))
			if math.Abs(float64(got.Origin.Y-y)) > 0.5 {
				t.Errorf("anchored row %d moved after its batch: layer y=%.1f; want y=%.1f", id, got.Origin.Y, y)
			}
		}
	})
}
