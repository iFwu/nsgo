package layout

import (
	"runtime"
	"testing"
	"weak"
)

// An out-of-flow box its containing block no longer has is freed: laid out
// again with fewer, the containing block's hoisted list keeps none of the
// others in its array past its end.
func TestDroppedHoistedBoxesAreFreed(t *testing.T) {
	abs := Style{Position: PositionAbsolute, Size: pxSize(10, 10)}
	kept := newLeaf(abs)
	cb := newNode(Style{Position: PositionRelative, Size: pxSize(100, 100)}, kept, newLeaf(abs), newLeaf(abs))
	dropped := []weak.Pointer[testNode]{weak.Make(cb.children[1]), weak.Make(cb.children[2])}
	computeLayout(cb, maxContentSpaces)
	if len(cb.state.Hoisted()) != 3 {
		t.Fatalf("hoisted %d boxes, want 3", len(cb.state.Hoisted()))
	}
	cb.children = []*testNode{kept}
	cb.markDirty()
	computeLayout(cb, maxContentSpaces)
	if h := cb.state.Hoisted(); len(h) != 1 || h[0] != Node(kept) {
		t.Fatalf("hoisted %d boxes, want the one left", len(h))
	}
	runtime.GC()
	for i, w := range dropped {
		if w.Value() != nil {
			t.Errorf("dropped box %d stays in memory", i)
		}
	}
	runtime.KeepAlive(cb)
}
