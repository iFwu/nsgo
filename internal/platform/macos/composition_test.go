//go:build darwin && !ios

package macos

import (
	"fmt"
	"testing"

	"github.com/ebitengine/purego/objc"

	"github.com/nsgo-dev/nsgo/internal/driver"
	"github.com/nsgo-dev/nsgo/internal/platform/apple"
)

// TestInputReportsWhereACompositionEnds has an input method end a composition of "r" after "ab" in each way one does
// through NSTextInputClient, in a single-line and a multiline input: the input reports the text the field holds after
// it and the composition's end. An input method that drops its composition, as one does on Escape, on Backspace over
// its last letter, or as the user switches input sources, marks nothing, which removes the marked text with no
// textDidChange.
func TestInputReportsWhereACompositionEnds(t *testing.T) {
	ends := []struct {
		name, want string
		caret      uint // where the composition leaves the caret in "r"
		end        func(tv objc.ID)
	}{
		{"marking nothing", "ab", 1, func(tv objc.ID) { markText(tv, "", 0) }},
		{"marking nothing, the caret before the composition", "ab", 0, func(tv objc.ID) { markText(tv, "", 0) }},
		{"marking nothing, then unmarking", "ab", 1, func(tv objc.ID) { markText(tv, "", 0); tv.Send(selUnmarkText) }},
		{"unmarking", "abr", 1, func(tv objc.ID) { tv.Send(selUnmarkText) }},
		{"inserting the letters", "abr", 1, func(tv objc.ID) { insertText(tv, "r") }},
		{"inserting a candidate", "ab日", 1, func(tv objc.ID) { insertText(tv, "日") }},
		{"inserting nothing", "ab", 1, func(tv objc.ID) { insertText(tv, "") }},
	}
	onMain(func() {
		loadAppKit()
		for _, multiline := range []bool{false, true} {
			for _, e := range ends {
				name := fmt.Sprintf("multiline %v, %s", multiline, e.name)
				s := &surface{views: make([]*view, 2), animator: driver.NewAnimator(func(driver.Event) {})}
				v := s.create(driver.KindInput, 1)
				s.views[v.id] = v
				window := objc.ID(keyWindowClass()).Send(selAlloc).Send(sel("initWithContentRect:styleMask:backing:defer:"),
					apple.Rect(0, 0, 300, 120), uint(0), uint(2), false)
				window.Send(sel("setReleasedWhenClosed:"), false)
				window.Send(sel("contentView")).Send(selAddSubview, v.native)
				v.native.Send(selSetFrame, apple.Rect(0, 0, 300, 120))
				v.configure(&driver.InputConfig{Multiline: multiline, Content: driver.Bounds{Size: driver.Size{Width: 300, Height: 120}}})
				in := v.input
				var events []driver.Event
				in.fs.Events = 1<<driver.EventChangeText | 1<<driver.EventCompositionChange
				in.fs.Dispatch = func(ev driver.Event) { events = append(events, ev) }
				in.value("ab", 0)
				window.Send(sel("makeFirstResponder:"), in.field)
				tv := in.editor()
				tv.Send(selSetSelRange, apple.NSRange{Location: 2})
				markText(tv, "r", e.caret)
				if in.text() != "abr" || in.fs.Known() != "abr" || !in.fs.Composing() {
					t.Errorf("%s: composing r shows %q, reported %q, composing %v; want abr reported and composing", name,
						in.text(), in.fs.Known(), in.fs.Composing())
				}
				events = nil
				e.end(tv)
				want := e.want
				if in.text() != want || in.fs.Known() != want || in.fs.Composing() {
					t.Errorf("%s: the field holds %q, reported %q, composing %v; want %q reported and the composition ended (events %+v)",
						name, in.text(), in.fs.Known(), in.fs.Composing(), want, events)
				}
				s.delete(v, false)
				window.Send(selRelease)
			}
		}
	})
}

// noRange is NSNotFound as a replacement range's location: the marked text's.
const noRange = uint(1<<63 - 1)

// markText has the first responder's text view tv mark text as an input method composing it does, the caret at caret
// in it.
func markText(tv objc.ID, text string, caret uint) {
	tv.Send(sel("setMarkedText:selectedRange:replacementRange:"), apple.NSString(text), apple.NSRange{Location: caret},
		apple.NSRange{Location: noRange})
}

// insertText has tv insert text over its marked text as an input method committing it does.
func insertText(tv objc.ID, text string) {
	tv.Send(sel("insertText:replacementRange:"), apple.NSString(text), apple.NSRange{Location: noRange})
}
