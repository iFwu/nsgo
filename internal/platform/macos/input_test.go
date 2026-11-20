//go:build darwin && !ios

package macos

import (
	"math"
	"strings"
	"sync"
	"testing"
	"unsafe"

	"github.com/ebitengine/purego/objc"

	"github.com/nsgo-dev/nsgo/internal/driver"
	"github.com/nsgo-dev/nsgo/internal/platform/apple"
)

// TestInputLayoutAfterUnmount lays out an input's native view, single-line and multiline, after the backend has removed it, as AppKit can for a view an outgoing transition still holds.
func TestInputLayoutAfterUnmount(t *testing.T) {
	onMain(func() {
		loadAppKit()
		s := &surface{views: make([]*view, 2), animator: driver.NewAnimator(func(driver.Event) {})}
		for _, multiline := range []bool{false, true} {
			v := s.create(driver.KindInput, 1)
			s.views[v.id] = v
			v.configure(&driver.InputConfig{Multiline: multiline, Content: driver.Bounds{Size: driver.Size{Width: 200, Height: 60}}})
			native := v.native.Send(sel("retain"))
			s.delete(v, false)
			if views[native] != nil {
				t.Error("the removed input is still registered")
			}
			native.Send(sel("layout"))
			native.Send(selRelease)
		}
	})
}

// TestMultilineEditorScrollsUnderPadding checks that a multiline input's
// scroll view fills the padding box and its text keeps the padding, uneven
// sides included (#701).
func TestMultilineEditorScrollsUnderPadding(t *testing.T) {
	onMain(func() {
		loadAppKit()
		s := &surface{views: make([]*view, 2), animator: driver.NewAnimator(func(driver.Event) {})}
		v := s.create(driver.KindInput, 1)
		s.views[v.id] = v
		v.native.Send(selSetFrame, apple.Rect(0, 0, 300, 200))
		pad := driver.Edges{Top: 18, Right: 20, Bottom: 20, Left: 12}
		v.configure(&driver.InputConfig{Multiline: true, Padding: pad,
			Content: driver.Bounds{Origin: driver.Point{X: 13, Y: 19}, Size: driver.Size{Width: 266, Height: 160}}})
		v.native.Send(sel("layout"))
		in := v.input
		if got, want := objc.Send[apple.CGRect](in.host, sel("frame")), apple.Rect(1, 1, 298, 198); got != want {
			t.Errorf("scroll view at %v, want the padding box %v", got, want)
		}
		if got := objc.Send[apple.CGPoint](in.field, sel("textContainerOrigin")); got != (apple.CGPoint{X: 12, Y: 18}) {
			t.Errorf("text starts at %v, want after the top and left padding", got)
		}
		in.host.Send(sel("tile"))
		visible := objc.Send[apple.CGSize](in.host, sel("contentSize")).Width
		wraps := objc.Send[apple.CGSize](in.field.Send(sel("textContainer")), sel("size")).Width
		if wraps != visible-32 {
			t.Errorf("text wraps at %v pt in a %v pt viewport, want the left and right padding off", wraps, visible)
		}
		s.delete(v, false)
	})
}

// TestMultilineEditorWrapsInItsViewport checks a multiline input's text
// view, its container and the text it lays out against the width the
// scroll view shows, as the input narrows and widens, with overlay and
// legacy scrollers (#42).
func TestMultilineEditorWrapsInItsViewport(t *testing.T) {
	onMain(func() { multilineEditorWraps(t) })
}

func multilineEditorWraps(t *testing.T) {
	loadAppKit()
	text := strings.Repeat("多行输入在可见区域内换行。", 20) + " https://example.com/" + strings.Repeat("a", 300)
	for _, style := range []int{1, 0} { // overlay, legacy
		window := objc.ID(apple.Class("NSWindow")).Send(selAlloc).Send(sel("initWithContentRect:styleMask:backing:defer:"),
			apple.Rect(0, 0, 500, 200), uint(0), uint(2), false)
		window.Send(sel("setReleasedWhenClosed:"), false)
		scroll, tv := newMultilineEditor()
		window.Send(sel("contentView")).Send(selAddSubview, scroll)
		scroll.Send(sel("setScrollerStyle:"), style)
		tv.Send(sel("setString:"), apple.NSString(text))
		container := tv.Send(sel("textContainer"))
		layout := tv.Send(sel("layoutManager"))
		for _, width := range []float64{320, 185, 445, 60} {
			scroll.Send(sel("setFrame:"), apple.Rect(0, 0, width, 90))
			scroll.Send(sel("tile"))
			layout.Send(sel("ensureLayoutForTextContainer:"), container)
			visible := objc.Send[apple.CGSize](scroll, sel("contentSize")).Width
			editor := objc.Send[apple.CGRect](tv, sel("frame")).Size.Width
			wraps := objc.Send[apple.CGSize](container, sel("size")).Width
			used := objc.Send[apple.CGRect](layout, sel("usedRectForTextContainer:"), container)
			if editor != visible || wraps != visible || used.Origin.X+used.Size.Width > visible {
				t.Errorf("scroller style %d, %v pt wide: shows %v pt, editor %v pt, container %v pt, text to %v pt",
					style, width, visible, editor, wraps, used.Origin.X+used.Size.Width)
			}
		}
		scroll.Send(selRelease)
		window.Send(selRelease)
	}
}

// TestMultilineEditorKeepsTheLineHeight checks that a multiline input
// draws its lines Style.LineHeight apart, as the engine lays the input
// out, each on the baseline a Text of its style puts it on: its
// placeholder, a value set over another with a line in a fallback font,
// lines that wrap and lines around one an input method marks, in
// TextKit 2 and in TextKit 1, which macOS 11 gives a text view.
func TestMultilineEditorKeepsTheLineHeight(t *testing.T) {
	onMain(func() {
		loadAppKit()
		style := driver.TextStyle{Font: driver.Font{Size: 14.5, Weight: 400}, Color: driver.RGB(0), LineHeight: 22}
		const width = 300
		baselines := func(text string) []float64 { return textBaselines(style, text, width) }
		for _, textKit1 := range []bool{false, true} {
			s := &surface{views: make([]*view, 2), animator: driver.NewAnimator(func(driver.Event) {})}
			v := s.create(driver.KindInput, 1)
			s.views[v.id] = v
			window := objc.ID(apple.Class("NSWindow")).Send(selAlloc).Send(sel("initWithContentRect:styleMask:backing:defer:"),
				apple.Rect(0, 0, width, 240), uint(0), uint(2), false)
			window.Send(sel("setReleasedWhenClosed:"), false)
			window.Send(sel("contentView")).Send(selAddSubview, v.native)
			v.native.Send(selSetFrame, apple.Rect(0, 0, width, 240))
			v.configure(&driver.InputConfig{Multiline: true, Style: style, Placeholder: "HHH",
				Content: driver.Bounds{Size: driver.Size{Width: width, Height: 240}}})
			in := v.input
			if textKit1 {
				in.field.Send(sel("layoutManager"))
			}
			// check compares where the ink of each line the input draws
			// ends, an H's on its baseline, with the baselines of a Text of
			// want: of the lines picked, by index into the Text's lines and
			// the runs of ink, where -1 is the last, or else of all.
			check := func(what, want string, lines ...[2]int) {
				t.Helper()
				v.native.Send(sel("layout"))
				bottoms, scale := inkBottoms(v.native)
				base := baselines(want)
				if lines == nil {
					for i := range min(len(bottoms), len(base)) {
						lines = append(lines, [2]int{i, i})
					}
				}
				for _, l := range lines {
					run := l[1]
					if run < 0 {
						run += len(bottoms)
					}
					if run < 0 || run >= len(bottoms) || l[0] >= len(base) || max(bottoms[run]-base[l[0]], base[l[0]]-bottoms[run]) > 1/scale {
						t.Errorf("TextKit 1 %v, %s: the lines end at %v, want a Text's baselines %v", textKit1, what, bottoms, base)
						return
					}
				}
			}
			check("placeholder", "HHH")
			in.value("HHH\n中文\nHHH", 0)
			check("a line in a fallback font", "HHH\n中文\nHHH", [2]int{0, 0}, [2]int{2, -1})
			long := strings.Repeat("HHHH ", 40)
			in.value(long, 0)
			check("lines that wrap", long, [2]int{0, 0}, [2]int{1, 1}, [2]int{2, 2}, [2]int{3, 3})
			in.value("HHH\n\nHHH", 0)
			in.field.Send(selSetSelRange, apple.NSRange{Location: 4})
			in.applying = true
			in.field.Send(sel("setMarkedText:selectedRange:replacementRange:"), apple.NSString("HHH"),
				apple.NSRange{Location: 3}, apple.NSRange{Location: 1<<63 - 1})
			in.applying = false
			check("lines around a marked one", "HHH\nHHH\nHHH", [2]int{0, 0}, [2]int{2, -1})
			s.delete(v, false)
			window.Send(selRelease)
		}
	})
}

// textBaselines returns the baselines of a Text of text in style, from
// its top, laid out width wide.
func textBaselines(style driver.TextStyle, text string, width float64) []float64 {
	p := apple.TextEngine{}.Prepare(&driver.AttributedText{Text: text, Style: style})
	defer p.Release()
	l := p.Layout(driver.Size{Width: driver.Points(width), Height: 1000}).(*apple.Layout)
	defer l.Release()
	var out []float64
	for _, line := range l.TextLines() {
		b := line.Bounds
		out = append(out, float64(b.Origin.Y+(b.Size.Height-line.Ascender-line.Descender)/2+line.Ascender))
	}
	return out
}

// keyWindowClass is a window that is always the key window, where a text
// view shows its caret, on screen or not.
var keyWindowClass = sync.OnceValue(func() objc.Class {
	return register("NsgoTestKeyWindow", apple.Class("NSWindow"), nil, []objc.MethodDef{
		{Cmd: sel("isKeyWindow"), Fn: func(objc.ID, objc.SEL) bool { return true }},
	})
})

// TestMultilineEditorCaretFitsTheFont checks that the lines of a multiline
// input taller than its font are as tall as the font where a Text puts
// them, not Style.LineHeight: its caret and the rect an input method puts
// its candidates by span a line's ascent and descent, on a line in the
// font and one in a fallback font. TextKit 1, which macOS 11 gives a text
// view, counts a line's spacing into the line, so there they start at the
// ascent and keep the full height.
func TestMultilineEditorCaretFitsTheFont(t *testing.T) {
	onMain(func() {
		loadAppKit()
		style := driver.TextStyle{Font: driver.Font{Size: 14.5, Weight: 400}, Color: driver.RGB(0), LineHeight: 22}
		const width, height, top = 300, 120, 5
		const text = "HHH\n中文\nHHH"
		base := textBaselines(style, text, width)
		for _, textKit1 := range []bool{false, true} {
			s := &surface{views: make([]*view, 2), animator: driver.NewAnimator(func(driver.Event) {})}
			v := s.create(driver.KindInput, 1)
			s.views[v.id] = v
			window := objc.ID(keyWindowClass()).Send(selAlloc).Send(sel("initWithContentRect:styleMask:backing:defer:"),
				apple.Rect(0, 0, width, height), uint(0), uint(2), false)
			window.Send(sel("setReleasedWhenClosed:"), false)
			window.Send(sel("contentView")).Send(selAddSubview, v.native)
			v.native.Send(selSetFrame, apple.Rect(0, 0, width, height))
			v.configure(&driver.InputConfig{Multiline: true, Style: style, Padding: driver.Edges{Top: top},
				Content: driver.Bounds{Origin: driver.Point{Y: top}, Size: driver.Size{Width: width, Height: height - top}}})
			in := v.input
			if textKit1 {
				in.field.Send(sel("layoutManager"))
			}
			in.value(text, 0)
			window.Send(sel("makeFirstResponder:"), in.field)
			font := in.field.Send(sel("font"))
			ascent, descent := objc.Send[float64](font, sel("ascender")), -objc.Send[float64](font, sel("descender"))
			indicator := apple.Class("NSTextInsertionIndicator") // macOS 14
			for i, loc := range []uint{1, 5, 9} {
				in.field.Send(selSetSelRange, apple.NSRange{Location: loc})
				v.native.Send(sel("layout"))
				want := top + base[i] - ascent
				fits := func(f apple.CGRect) bool {
					return math.Abs(f.Origin.Y-want) <= 0.5 && (textKit1 || math.Abs(f.Size.Height-(ascent+descent)) <= 0.5)
				}
				r := objc.Send[apple.CGRect](in.field, sel("firstRectForCharacterRange:actualRange:"), apple.NSRange{Location: loc, Length: 1}, uintptr(0))
				r = objc.Send[apple.CGRect](window, sel("convertRectFromScreen:"), r)
				r = objc.Send[apple.CGRect](in.field, sel("convertRect:fromView:"), r, objc.ID(0))
				if !fits(r) {
					t.Errorf("TextKit 1 %v, line %d: an input method's rect is %.2f pt tall at %.2f, want the font's %.2f at %.2f",
						textKit1, i, r.Size.Height, r.Origin.Y, ascent+descent, want)
				}
				if indicator == 0 {
					continue
				}
				for _, sub := range apple.Subviews(in.field) {
					if objc.Send[bool](sub, sel("isKindOfClass:"), indicator) {
						if f := objc.Send[apple.CGRect](sub, sel("frame")); !fits(f) {
							t.Errorf("TextKit 1 %v, line %d: the caret is %.2f pt tall at %.2f, want the font's %.2f at %.2f",
								textKit1, i, f.Size.Height, f.Origin.Y, ascent+descent, want)
						}
					}
				}
			}
			s.delete(v, false)
			window.Send(selRelease)
		}
	})
}

// inkBottoms returns where each run of rows holding dark ink in view ends,
// from its top, and the scale it draws at.
func inkBottoms(view objc.ID) (bottoms []float64, scale float64) {
	r := objc.Send[apple.CGRect](view, sel("bounds"))
	rep := view.Send(sel("bitmapImageRepForCachingDisplayInRect:"), r)
	view.Send(sel("cacheDisplayInRect:toBitmapImageRep:"), r, rep)
	data := unsafe.Slice((*byte)(objc.Send[unsafe.Pointer](rep, sel("bitmapData"))), objc.Send[int](rep, sel("bytesPerRow"))*objc.Send[int](rep, sel("pixelsHigh")))
	row, step := objc.Send[int](rep, sel("bytesPerRow")), objc.Send[int](rep, sel("samplesPerPixel"))
	w, h := objc.Send[int](rep, sel("pixelsWide")), objc.Send[int](rep, sel("pixelsHigh"))
	scale = float64(h) / r.Size.Height
	inked := false
	for y := range h + 1 {
		dark := false
		for x := 0; y < h && x < w && !dark; x++ {
			p := data[y*row+x*step:]
			dark = p[3] > 128 && p[0] < 128
		}
		if inked && !dark {
			bottoms = append(bottoms, float64(y)/scale)
		}
		inked = dark
	}
	return bottoms, scale
}
