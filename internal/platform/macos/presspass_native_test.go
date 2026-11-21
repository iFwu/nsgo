//go:build darwin && !ios

package macos

import (
	"testing"

	"github.com/ebitengine/purego/objc"

	"github.com/nsgo-dev/nsgo/internal/driver"
	"github.com/nsgo-dev/nsgo/internal/platform/apple"
)

// A press on a part of a popover that takes no press goes on up the popover's views, never to the view behind it, where
// AppKit's own mouseDown: hands it.
func TestUnansweredPressStaysOffTheViewsBehind(t *testing.T) {
	onMain(func() {
		loadAppKit()
		window := objc.ID(apple.Class("NSWindow")).Send(selAlloc).Send(sel("initWithContentRect:styleMask:backing:defer:"),
			apple.Rect(0, 0, 400, 300), uint(0), uint(2), false)
		defer window.Send(selRelease)
		s := &surface{window: window}
		makeView := func(parent *view, f driver.Bounds) *view {
			n := objc.ID(classView).Send(selNew)
			n.Send(selSetWantsLayer, true)
			v := &view{native: n, content: n, surface: s, kind: driver.KindView, up: parent, frame: f, framed: true,
				style: driver.ViewStyle{Opacity: 1}}
			views[n] = v
			n.Send(selSetFrame, apple.Rect(float64(f.Origin.X), float64(f.Origin.Y), float64(f.Size.Width), float64(f.Size.Height)))
			if parent != nil {
				parent.native.Send(selAddSubview, n)
			}
			t.Cleanup(func() { onMain(func() { delete(views, n); n.Send(selRelease) }) })
			return v
		}
		root := makeView(nil, driver.Bounds{Size: driver.Size{Width: 400, Height: 300}})
		s.native = root.native
		window.Send(sel("setContentView:"), root.native)
		row := makeView(root, driver.Bounds{Size: driver.Size{Width: 400, Height: 300}})
		row.events = driver.EventMask(1) << driver.EventPress
		popover := makeView(root, driver.Bounds{Origin: driver.Point{X: 50, Y: 50}, Size: driver.Size{Width: 300, Height: 200}})
		label := makeView(popover, driver.Bounds{Origin: driver.Point{X: 20, Y: 20}, Size: driver.Size{Width: 100, Height: 20}})

		// The label's middle in the root's coordinates, which go down from its top, and in the window's, which go up.
		mid := apple.CGPoint{X: 50 + 20 + 50, Y: 50 + 20 + 10}
		at := apple.CGPoint{X: mid.X, Y: 300 - mid.Y}
		if hit := root.native.Send(sel("hitTest:"), mid); hit != label.native {
			t.Fatalf("the press lands on %s, want the label", apple.GoString(hit.Send(sel("className"))))
		}
		down := objc.ID(apple.Class("NSEvent")).Send(
			sel("mouseEventWithType:location:modifierFlags:timestamp:windowNumber:context:eventNumber:clickCount:pressure:"),
			uint(evLeftDown), at, uint(0), float64(0), objc.Send[int](window, sel("windowNumber")), objc.ID(0), 0, 1, float32(1))
		label.native.Send(sel("mouseDown:"), down)
		if row.pressing.Load() {
			t.Fatal("a press on the popover's label went to the view behind it")
		}
	})
}
