package imtermbox

import (
	"github.com/andyleap/imterm"

	tb "github.com/nsf/termbox-go"
)

type TermAdapter struct {
}

func (ta *TermAdapter) SetCell(x, y int, ch rune, fg, bg imterm.Attribute) {
	tb.SetCell(x, y, ch, tb.Attribute(fg), tb.Attribute(bg))
}
func (ta *TermAdapter) Size() (w, h int) {
	return tb.Size()
}
func (ta *TermAdapter) Flip() {
	tb.Flush()
}
func (ta *TermAdapter) Clear(bg imterm.Attribute) {
	tb.Clear(tb.ColorDefault, tb.Attribute(bg))
}
