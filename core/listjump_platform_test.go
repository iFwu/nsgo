package core_test

import (
	"fmt"
	"testing"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/internal/driver"
	"github.com/nsgo-dev/nsgo/nsgotest"
)

// viewportGap reports the largest run of the list's viewport no mounted
// cell covers, in points, and the rows missing there.
func viewportGap(a *nsgotest.App, list *nsgotest.View, rows int) (core.Points, []int) {
	top := list.Offset.Y
	bottom := top + list.Frame.Size.Height
	type span struct{ lo, hi core.Points }
	var spans []span
	for i := 0; i < rows; i++ {
		vs := a.Query(fmt.Sprintf("text:r%03d", i))
		if len(vs) == 0 {
			continue
		}
		// The positioned frame is the list cell, the row view's parent.
		f := vs[0].Parent.Parent.Frame
		spans = append(spans, span{f.Origin.Y, f.Origin.Y + f.Size.Height})
	}
	// Sort by top; find coverage of [top,bottom].
	for i := 0; i < len(spans); i++ {
		for j := i + 1; j < len(spans); j++ {
			if spans[j].lo < spans[i].lo {
				spans[i], spans[j] = spans[j], spans[i]
			}
		}
	}
	var worst core.Points
	var missing []int
	cursor := top
	for _, s := range spans {
		if s.hi <= top || s.lo >= bottom {
			continue
		}
		if s.lo > cursor {
			if d := s.lo - cursor; d > worst {
				worst = d
			}
			missing = append(missing, -1)
		}
		if s.hi > cursor {
			cursor = s.hi
		}
		if cursor >= bottom {
			break
		}
	}
	if bottom > cursor {
		if d := bottom - cursor; d > worst {
			worst = d
		}
		missing = append(missing, -1)
	}
	return worst, missing
}

// A conversation switch as the client makes it, stepped a frame at a
// time: every row's key changes, the count shrinks and the list scrolls
// to its end in one batch. Each step reports the viewport's largest
// uncovered band.
func TestListJumpToEndCoverageSteps(t *testing.T) {
	for _, tc := range []struct {
		name    string
		count   int
		newRows int
		height  func(i int) core.Points
	}{
		{"uniform-27", 200, 120, func(i int) core.Points { return 27 }},
		{"mixed", 200, 120, func(i int) core.Points {
			if i%9 == 0 {
				return 240
			}
			return 27
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var conv *core.Signal[int]
			var ref *core.Ref
			a := nsgotest.Start(t, func(cx *core.Scope) core.Node {
				conv = core.NewSignal(cx, 0)
				ref = &core.Ref{}
				return core.List{
					Ref:    ref,
					TestID: "list",
					Style:  core.Style{Height: core.Pt(800)},
					Count: func() int {
						if conv.Get() == 0 {
							return tc.count
						}
						return tc.newRows
					},
					Key:           func(i int) any { return fmt.Sprintf("c%d-%d", conv.Get(), i) },
					EstimatedSize: 80,
					Row: func(cx *core.Scope, row core.ListRow) core.Node {
						return core.View{StyleFunc: func() core.Style {
							return core.Style{Height: core.Pt(tc.height(row.Index()))}
						}, Children: []core.Node{
							core.Text{ContentFunc: func() string { return fmt.Sprintf("r%03d", row.Index()) }},
						}}
					},
				}
			}, nsgotest.ManualFrames())
			a.Settle()
			a.Post(func() {
				conv.Set(1)
				ref.ScrollToEnd(false)
			})
			// One settle for the jump's own commit, then a frame at a time.
			a.Settle()
			for step := 0; step < 12; step++ {
				list := a.Find("id:list")
				gap, _ := viewportGap(a, list, tc.newRows)
				t.Logf("step %d: offset %v content %v viewport-gap %v", step, list.Offset.Y, list.Content.Height, gap)
				a.Advance(driver.FrameInterval)
			}
			list := a.Find("id:list")
			gap, _ := viewportGap(a, list, tc.newRows)
			t.Logf("final: offset %v content %v viewport-gap %v", list.Offset.Y, list.Content.Height, gap)
			if gap != 0 {
				t.Fatalf("viewport has a %v uncovered band after settling", gap)
			}
		})
	}
}
