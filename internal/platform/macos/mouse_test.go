//go:build darwin && !ios

package macos

import (
	"fmt"
	"testing"

	"github.com/ebitengine/purego/objc"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/internal/driver"
	"github.com/nsgo-dev/nsgo/internal/engine"
	"github.com/nsgo-dev/nsgo/internal/platform/apple"
)

// The window server counts a quick click where a pushed page has replaced the view that took the one before as a double click; the window makes it the new view's first click, and counts that view's clicks after it from there (#900).
func TestClicksRestartOnAnotherView(t *testing.T) {
	onMain(func() {
		loadAppKit()
		pool := objc.ID(apple.Class("NSAutoreleasePool")).Send(selNew)
		defer pool.Send(sel("drain"))
		window := objc.ID(classWindow).Send(selAlloc).Send(sel("initWithContentRect:styleMask:backing:defer:"),
			apple.Rect(0, 0, 200, 100), uint(0), uint(2), false)
		window.Send(sel("setReleasedWhenClosed:"), false)
		s := newSurface(Options{Width: 200, Height: 100}, false)
		s.window = window
		window.Send(sel("setContentView:"), s.native)
		defer func() {
			s.close()
			delete(views, s.native)
			s.native.Send(selRelease)
			window.Send(sel("close"))
			window.Send(selRelease)
		}()
		for _, x := range []float64{0, 100} {
			side := objc.ID(apple.Class("NSView")).Send(selAlloc).Send(sel("initWithFrame:"), apple.Rect(x, 0, 100, 100))
			s.native.Send(sel("addSubview:"), side)
			side.Send(selRelease)
		}
		const left, right = 50, 150
		for i, c := range []struct {
			kind      uint
			x         float64
			reported  int
			delivered int
		}{
			{evLeftDown, left, 1, 1},
			{evLeftDown, right, 2, 1}, // the page changed under the pointer
			{evLeftUp, right, 2, 2},
			{evLeftDown, right, 3, 2}, // a double click of the new view
			{evLeftDown, right, 1, 1},
			{evLeftDown, right, 2, 2}, // a double click on one view stays one
			{evLeftDown, left, 3, 1},
		} {
			event := objc.ID(apple.Class("NSEvent")).Send(
				sel("mouseEventWithType:location:modifierFlags:timestamp:windowNumber:context:eventNumber:clickCount:pressure:"),
				c.kind, apple.CGPoint{X: c.x, Y: 50}, uint(0), float64(i), objc.Send[int](window, sel("windowNumber")), objc.ID(0), 0, c.reported, float32(1))
			if got := objc.Send[int](s.restartClicks(event), sel("clickCount")); got != c.delivered {
				t.Errorf("event %d at %v: clickCount %d, want %d", i, c.x, got, c.delivered)
			}
		}
	})
}

// A view drawn outside its parent, as a card floating out of a collapsed sidebar's zero-width slot is, hears the
// pointer come and go where the hit test finds it: AppKit sends its tracking area no entry there.
func TestWindowHoversViewOutsideItsParent(t *testing.T) {
	onMain(func() {
		loadAppKit()
		pool := objc.ID(apple.Class("NSAutoreleasePool")).Send(selNew)
		defer pool.Send(sel("drain"))
		var heard []driver.EventKind
		eng := engine.Start(actionEngine{}, func(*core.Scope) core.Node { return core.View{} }, engine.Options{Event: func(_ uint32, e driver.Event) {
			if e.Kind == driver.EventMouseEnter || e.Kind == driver.EventMouseLeave {
				heard = append(heard, e.Kind)
			}
		}})
		defer eng.Close()
		window := objc.ID(classWindow).Send(selAlloc).Send(sel("initWithContentRect:styleMask:backing:defer:"),
			apple.Rect(0, 0, 300, 200), uint(0), uint(2), false)
		window.Send(sel("setReleasedWhenClosed:"), false)
		s := newSurface(Options{Width: 300, Height: 200}, false)
		s.engine = eng
		s.views = make([]*view, 3)
		s.animator = driver.NewAnimator(func(driver.Event) {})
		s.window = window
		window.Send(sel("setContentView:"), s.native)
		defer func() {
			s.close()
			delete(views, s.native)
			s.native.Send(selRelease)
			window.Send(sel("close"))
			window.Send(selRelease)
		}()

		slot := s.create(driver.KindView, 1)
		s.views[1] = slot
		slot.native.Send(selSetFrame, apple.Rect(0, 0, 0, 200))
		slot.frame, slot.framed = driver.Bounds{Size: driver.Size{Height: 200}}, true
		slot.overflow = driver.Bounds{Size: driver.Size{Width: 120, Height: 200}}
		s.native.Send(selAddSubview, slot.native)
		card := s.create(driver.KindView, 2)
		s.views[2], card.up = card, slot
		card.native.Send(selSetFrame, apple.Rect(0, 0, 120, 200))
		card.frame, card.framed = driver.Bounds{Size: driver.Size{Width: 120, Height: 200}}, true
		slot.childHost().Send(selAddSubview, card.native)
		card.setEvents(1<<driver.EventMouseEnter | 1<<driver.EventMouseLeave)
		defer func() {
			s.delete(card, false)
			s.delete(slot, false)
		}()

		toWin := func(x, y float64) apple.CGPoint {
			return objc.Send[apple.CGPoint](s.native, sel("convertPoint:toView:"), apple.CGPoint{X: x, Y: y}, objc.ID(0))
		}
		on, off := toWin(60, 100), toWin(200, 100)
		if !covers(card, s.hitAt(on)) {
			t.Fatal("the hit test misses the card outside its slot")
		}
		s.pick(on, true, true)
		if fmt.Sprint(heard) != fmt.Sprint([]driver.EventKind{driver.EventMouseEnter}) {
			t.Fatalf("the pointer coming onto the card reported %v, want an enter", heard)
		}
		s.pick(off, true, true)
		if fmt.Sprint(heard) != fmt.Sprint([]driver.EventKind{driver.EventMouseEnter, driver.EventMouseLeave}) {
			t.Fatalf("the pointer leaving the card reported %v, want an enter and a leave", heard)
		}
		if _, in := s.inside[card]; in || card.over {
			t.Fatal("the card stayed hovered once the pointer left it")
		}
	})
}
