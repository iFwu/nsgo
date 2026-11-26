package core

import (
	"fmt"
	"testing"

	"github.com/nsgo-dev/nsgo/internal/driver"
)

// settleHost runs the display frames and commits a quiet tree needs, as a
// platform's frames do for the work a commit paces.
func settleHost(h *host) {
	for {
		h.drain()
		h.frame()
		if h.tree.Commit() == nil && !h.tree.Pacing() {
			return
		}
	}
}

// viewportRows reports the rows list ln's viewport shows: the ones with a
// bound cell and the ones without, in index order, with the span it covers
// and the content's length.
func viewportRows(ln *listNode) (missing, bound []int, top, bottom, content Points) {
	st := &ln.st
	top = st.naturalOffset()
	bottom, content = top+st.viewport, st.contentLen()
	first, last, ok := st.itemsIn(top, bottom)
	if !ok {
		return nil, nil, top, bottom, content
	}
	for i := max(first, 0); i <= min(last, st.count-1); i++ {
		if _, has := ln.byIndex[i]; has {
			bound = append(bound, i)
		} else {
			missing = append(missing, i)
		}
	}
	return missing, bound, top, bottom, content
}

// commitsToCover counts the commits after the jump's until the viewport's
// rows are all bound, at most max.
func commitsToCover(h *host, ln *listNode, max int) int {
	for i := 0; i < max; i++ {
		missing, _, _, _, _ := viewportRows(ln)
		if len(missing) == 0 {
			return i
		}
		h.frame()
		h.commit()
	}
	return max
}

// lastStyle is the view style batch b last sent for node id, nil without one.
func lastStyle(b *driver.MountBatch, id NodeID) *driver.ViewStyle {
	var vs *driver.ViewStyle
	for _, it := range b.Items {
		if it.Kind == driver.ItemViewStyle && it.ID == id {
			vs = it.ViewStyle
		}
	}
	return vs
}

// listFrameAudit checks the spare cells' emitted display, duplicate cells
// for one row, and cells placed outside the content, reporting what is wrong.
func listFrameAudit(h *host, ln *listNode, b *driver.MountBatch) []string {
	var bad []string
	st := &ln.st
	content := st.contentLen()
	// Spare cells must be hidden in what the commit sent.
	for _, c := range ln.free {
		if c.node.base.Display != DisplayNone {
			bad = append(bad, fmt.Sprintf("spare cell of row %d has Display %v in the model", c.index, c.node.base.Display))
		}
		if vs := lastStyle(b, c.node.id); vs == nil || !vs.Hidden {
			bad = append(bad, fmt.Sprintf("spare cell of row %d sent no hidden style (last %v)", c.index, vs))
		}
	}
	// One cell per row, inside the content.
	seen := map[int]int{}
	for i, c := range ln.byIndex {
		seen[c.index]++
		at := st.itemAt(i)
		if at < -pinSlack || at+st.rowExtent(i) > content+pinSlack {
			bad = append(bad, fmt.Sprintf("cell of row %d placed at %v..%v outside content %v", i, at, at+st.rowExtent(i), content))
		}
		if c.node.frame.Origin.Y < -pinSlack || c.node.frame.Origin.Y+c.node.frame.Size.Height > content+pinSlack {
			bad = append(bad, fmt.Sprintf("cell of row %d frame %v outside content %v", i, c.node.frame, content))
		}
	}
	for i, n := range seen {
		if n > 1 {
			bad = append(bad, fmt.Sprintf("row %d has %d cells", i, n))
		}
	}
	return bad
}

// A jump to the end binds the rows the viewport shows in the commit that
// makes it, estimates far from the rows' measured sizes aside: rows left
// unbound show as a blank band until later commits bind them.
func TestListCoversViewportInTheCommitThatJumpsToItsEnd(t *testing.T) {
	for _, viewport := range []Points{600, 800, 900} {
		t.Run(fmt.Sprintf("viewport=%v", viewport), func(t *testing.T) {
			ref := &Ref{}
			h := newHost(t, Size{Width: 200, Height: viewport}, func(cx *Scope) Node {
				return List{Ref: ref, TestID: "list", Style: Style{Height: Pt(viewport)},
					Count: func() int { return 200 }, EstimatedSize: 80,
					Row: func(*Scope, ListRow) Node { return View{Style: Style{Height: Pt(27)}} }}
			})
			settleHost(h)
			ref.ScrollToEnd(false)
			b := h.commit()
			ln := h.find("list").list
			missing, bound, top, bottom, content := viewportRows(ln)
			t.Logf("jump commit: items %d, offset %v, viewport %v, content %v; viewport rows bound %d (%v..%v), missing %v, cells %d",
				len(b.Items), top, bottom-top, content, len(bound), bound[:min(3, len(bound))], bound[max(0, len(bound)-3):], missing, len(ln.byIndex))
			for _, bad := range listFrameAudit(h, ln, b) {
				t.Logf("audit: %s", bad)
			}
			if len(missing) > 0 {
				t.Errorf("%d viewport rows unbound after the jump commit: %v", len(missing), missing[:min(10, len(missing))])
			}
			t.Logf("viewport covered after %d more commits", commitsToCover(h, ln, 30))
		})
	}
}

// The same jump as an app makes switching conversations: every row's key
// changes and the count shrinks in the batch that scrolls to the end.
func TestListCoversViewportWhenItsRowsAreReplacedAndItJumpsToItsEnd(t *testing.T) {
	for _, viewport := range []Points{600, 800, 900} {
		t.Run(fmt.Sprintf("viewport=%v", viewport), func(t *testing.T) {
			var conv *Signal[int]
			ref := &Ref{}
			h := newHost(t, Size{Width: 200, Height: viewport}, func(cx *Scope) Node {
				conv = NewSignal(cx, 0)
				return List{Ref: ref, TestID: "list", Style: Style{Height: Pt(viewport)},
					Count: func() int {
						if conv.Get() == 0 {
							return 200
						}
						return 120
					},
					Key:           func(i int) any { return fmt.Sprintf("c%d-%d", conv.Get(), i) },
					EstimatedSize: 80,
					Row:           func(*Scope, ListRow) Node { return View{Style: Style{Height: Pt(27)}} }}
			})
			settleHost(h)
			conv.Set(1)
			ref.ScrollToEnd(false)
			b := h.commit()
			ln := h.find("list").list
			missing, bound, top, bottom, content := viewportRows(ln)
			t.Logf("replaced jump commit: items %d, offset %v, viewport %v, content %v; viewport rows bound %d (%v..%v), missing %v, cells %d, free %d",
				len(b.Items), top, bottom-top, content, len(bound), bound[:min(3, len(bound))], bound[max(0, len(bound)-3):], missing, len(ln.byIndex), len(ln.free))
			for _, bad := range listFrameAudit(h, ln, b) {
				t.Logf("audit: %s", bad)
			}
			if len(missing) > 0 {
				t.Errorf("%d viewport rows unbound after the replaced jump commit: %v", len(missing), missing[:min(10, len(missing))])
			}
			t.Logf("viewport covered after %d more commits", commitsToCover(h, ln, 30))
		})
	}
}
