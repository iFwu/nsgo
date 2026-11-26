package core

import (
	"fmt"
	"testing"

	"github.com/nsgo-dev/nsgo/internal/driver"
)

// platformScrolls applies batch b's scroll items for list ln's scroll view
// as a platform would: each ItemScrollTo clamped to the content, each
// ItemScrollBy added, each landing reported back at once, which is what
// moves the list's offset and lead on a real backend.
func platformScrolls(h *host, ln *listNode, b *driver.MountBatch, platformOff *Points, applied *uint32) {
	v := ln.view
	if v == nil || v.scroll == nil {
		return
	}
	content := ln.st.contentLen()
	viewport := ln.st.viewport
	for _, it := range b.Items {
		if it.ID != v.id {
			continue
		}
		switch it.Kind {
		case driver.ItemScrollTo:
			lo, hi := Points(0), max(content-viewport, 0)
			*platformOff = min(max(Points(it.Point.Y), lo), hi)
		case driver.ItemScrollBy:
			*platformOff += it.Point.Y
		default:
			continue
		}
		*applied++
		lo, hi := Points(0), max(content-viewport, 0)
		*platformOff = min(max(*platformOff, lo), hi)
		h.tree.Dispatch(driver.Event{Kind: driver.EventScroll, ID: v.id, Point: driver.Point{Y: *platformOff}, Count: *applied})
	}
	// A platform clamps an offset past its shrunken content and reports it,
	// with or without a scroll of the app's own in the batch.
	lo, hi := Points(0), max(content-viewport, 0)
	clamped := min(max(*platformOff, lo), hi)
	if clamped == *platformOff {
		return
	}
	*platformOff = clamped
	h.tree.Dispatch(driver.Event{Kind: driver.EventScroll, ID: v.id, Point: driver.Point{Y: *platformOff}, Count: *applied})
}

// viewportBand is the largest run of the viewport's rows no bound cell
// covers, measured on the cells' real frames; the padding past the last
// row is meant to be blank and does not count.
func viewportBand(ln *listNode, platformOff Points) (Points, []int) {
	st := &ln.st
	b := st.blocks()
	top, bottom := max(platformOff, b.body), min(platformOff+st.viewport, b.body+st.itemsLen())
	if bottom <= top {
		return 0, nil
	}
	type span struct{ lo, hi Points }
	var spans []span
	for _, c := range ln.byIndex {
		f := c.node.frame
		spans = append(spans, span{f.Origin.Y, f.Origin.Y + ln.main(f.Size)})
	}
	for i := 0; i < len(spans); i++ {
		for j := i + 1; j < len(spans); j++ {
			if spans[j].lo < spans[i].lo {
				spans[i], spans[j] = spans[j], spans[i]
			}
		}
	}
	var worst Points
	cursor := top
	for _, s := range spans {
		if s.hi <= top || s.lo >= bottom {
			continue
		}
		if s.lo > cursor && s.lo-cursor > worst {
			worst = s.lo - cursor
		}
		if s.hi > cursor {
			cursor = s.hi
		}
		if cursor >= bottom {
			return worst, nil
		}
	}
	if d := bottom - cursor; d > worst {
		worst = d
	}
	return worst, nil
}

// The jump commit's own passes are not the whole story on a backend that
// reports scrolls: each report commits again while the list settles, and
// the client's reveal check asks for the end again until it lands. This
// steps those commits the way the platform drives them and checks the
// viewport's coverage on the cells' real frames. A jump can leave the
// viewport uncovered for the commit-cycle its report lands in; what must
// not happen is the band staying or coming back commit after commit.
func TestListJumpToEndCoverageWithPlatformReports(t *testing.T) {
	for _, tc := range []struct {
		name       string
		count      int
		newRows    int
		viewport   Points
		height     func(i int) Points
		pageSize   int  // rows of the switched-to list delivered per commit, 0 for all at once
		streamTail bool // the last quarter of rows keeps growing, as a reply streams
	}{
		{"uniform-27", 200, 120, 800, func(i int) Points { return 27 }, 0, false},
		{"uniform-27-big", 800, 500, 800, func(i int) Points { return 27 }, 0, false},
		{"tiny-rows", 400, 250, 800, func(i int) Points { return 12 }, 0, false},
		{"tiny-rows-short", 400, 250, 400, func(i int) Points { return 12 }, 0, false},
		{"mixed", 200, 120, 800, func(i int) Points {
			if i%9 == 0 {
				return 240
			}
			return 27
		}, 0, false},
		{"mixed-big", 800, 500, 800, func(i int) Points {
			if i%9 == 0 {
				return 240
			}
			return 27
		}, 0, false},
		{"arriving", 200, 120, 800, func(i int) Points { return 27 }, 50, false},
		{"streaming", 200, 120, 800, func(i int) Points { return 27 }, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var conv, arrive, tick *Signal[int]
			ref := &Ref{}
			h := newHost(t, Size{Width: 200, Height: tc.viewport}, func(cx *Scope) Node {
				conv = NewSignal(cx, 0)
				arrive = NewSignal(cx, 0)
				tick = NewSignal(cx, 0)
				return List{Ref: ref, TestID: "list", Style: Style{Height: Pt(tc.viewport)},
					Count: func() int {
						if conv.Get() == 0 {
							return tc.count
						}
						if tc.pageSize > 0 {
							return arrive.Get()
						}
						return tc.newRows
					},
					Key:           func(i int) any { return fmt.Sprintf("c%d-%d", conv.Get(), i) },
					EstimatedSize: 80,
					Header:        View{Style: Style{Height: Pt(40)}},
					Row: func(cx *Scope, row ListRow) Node {
						return View{StyleFunc: func() Style {
							h := tc.height(row.Index())
							if tc.streamTail && row.Index() >= tc.newRows*3/4 {
								h += Points(min(tick.Get(), 8) * 6)
							}
							return Style{Height: Pt(h)}
						}}
					}}
			})
			settleHost(h)
			conv.Set(1)
			ref.ScrollToEnd(false)
			ln := h.find("list").list
			var platformOff Points
			// Scrolls the settle commits sent all landed; count them applied.
			applied := ln.view.scroll.issued
			uncovered := 0
			for step := 0; step < 30; step++ {
				// The client's reveal check re-asks for the end until the
				// list lands and stays: one ScrollToEnd per commit.
				if step > 0 {
					ref.ScrollToEnd(false)
				}
				if tc.pageSize > 0 {
					// Pages of the switched-to transcript land over commits.
					arrive.Set(min((step+1)*tc.pageSize, tc.newRows))
				}
				if tc.streamTail {
					tick.Set(step)
				}
				b := h.commit()
				platformScrolls(h, ln, b, &platformOff, &applied)
				missing, _, top, _, content := viewportRows(ln)
				band, _ := viewportBand(ln, platformOff)
				// H2: no spare cell may stay visible; their hidden style was
				// emitted when they were freed, so the model state is the check.
				// And no bound cell may hang past the content's end, where a
				// phantom row would draw below the last one.
				blocks := ln.st.blocks()
				rowsEnd := blocks.body + ln.st.itemsLen()
				for _, c := range ln.free {
					if c.node.base.Display != DisplayNone {
						t.Errorf("commit %d: spare cell of row %d has Display %v", step, c.index, c.node.base.Display)
					}
				}
				for i, c := range ln.byIndex {
					f := c.node.frame
					if f.Origin.Y+ln.main(f.Size) > rowsEnd+pinSlack {
						t.Errorf("commit %d: cell of row %d hangs past the rows' end %v (frame %v)", step, i, rowsEnd, f)
					}
				}
				t.Logf("commit %d: offset %v platform %v content %v missing %v band %v cells %d waiting %d",
					step, top, platformOff, content, missing[:min(8, len(missing))], band, len(ln.byIndex), len(ln.waiting))
				if len(missing) == 0 && band == 0 {
					uncovered = 0
				} else {
					uncovered++
					if uncovered > 2 {
						t.Fatalf("commit %d: viewport uncovered for %d commits running", step, uncovered)
					}
				}
				if uncovered == 0 && h.commit() == nil {
					break
				}
			}
			if uncovered != 0 {
				t.Fatalf("viewport still uncovered after 30 commits")
			}
		})
	}
}
