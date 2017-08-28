package imterm

import (
	"strings"

	"github.com/mitchellh/go-wordwrap"
)

type Attribute uint16

// Cell colors, you can combine a color with multiple attributes using bitwise
// OR ('|').
const (
	ColorDefault Attribute = iota
	ColorBlack
	ColorRed
	ColorGreen
	ColorYellow
	ColorBlue
	ColorMagenta
	ColorCyan
	ColorWhite
)

// Cell attributes, it is possible to use multiple attributes by combining them
// using bitwise OR ('|'). Although, colors cannot be combined. But you can
// combine attributes and a single color.
//
// It's worth mentioning that some platforms don't support certain attibutes.
// For example windows console doesn't support AttrUnderline. And on some
// terminals applying AttrBold to background may result in blinking text. Use
// them with caution and test your code on various terminals.
const (
	AttrBold Attribute = 1 << (iota + 9)
	AttrUnderline
	AttrReverse
)

type Key uint16

const (
	KeyF1 Key = 0xFFFF - iota
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12
	KeyInsert
	KeyDelete
	KeyHome
	KeyEnd
	KeyPgup
	KeyPgdn
	KeyArrowUp
	KeyArrowDown
	KeyArrowLeft
	KeyArrowRight
	key_min // see terminfo
)

type MouseButton int

const (
	MouseNone MouseButton = iota
	MouseLeft
	MouseRight
	MouseMiddle
	MouseWheelUp
	MouseWheelDown
	MouseRelease
)

const (
	KeyCtrlTilde      Key = 0x00
	KeyCtrl2          Key = 0x00
	KeyCtrlSpace      Key = 0x00
	KeyCtrlA          Key = 0x01
	KeyCtrlB          Key = 0x02
	KeyCtrlC          Key = 0x03
	KeyCtrlD          Key = 0x04
	KeyCtrlE          Key = 0x05
	KeyCtrlF          Key = 0x06
	KeyCtrlG          Key = 0x07
	KeyBackspace      Key = 0x08
	KeyCtrlH          Key = 0x08
	KeyTab            Key = 0x09
	KeyCtrlI          Key = 0x09
	KeyCtrlJ          Key = 0x0A
	KeyCtrlK          Key = 0x0B
	KeyCtrlL          Key = 0x0C
	KeyEnter          Key = 0x0D
	KeyCtrlM          Key = 0x0D
	KeyCtrlN          Key = 0x0E
	KeyCtrlO          Key = 0x0F
	KeyCtrlP          Key = 0x10
	KeyCtrlQ          Key = 0x11
	KeyCtrlR          Key = 0x12
	KeyCtrlS          Key = 0x13
	KeyCtrlT          Key = 0x14
	KeyCtrlU          Key = 0x15
	KeyCtrlV          Key = 0x16
	KeyCtrlW          Key = 0x17
	KeyCtrlX          Key = 0x18
	KeyCtrlY          Key = 0x19
	KeyCtrlZ          Key = 0x1A
	KeyEsc            Key = 0x1B
	KeyCtrlLsqBracket Key = 0x1B
	KeyCtrl3          Key = 0x1B
	KeyCtrl4          Key = 0x1C
	KeyCtrlBackslash  Key = 0x1C
	KeyCtrl5          Key = 0x1D
	KeyCtrlRsqBracket Key = 0x1D
	KeyCtrl6          Key = 0x1E
	KeyCtrl7          Key = 0x1F
	KeyCtrlSlash      Key = 0x1F
	KeyCtrlUnderscore Key = 0x1F
	KeySpace          Key = 0x20
	KeyBackspace2     Key = 0x7F
	KeyCtrl8          Key = 0x7F
)

type Screen interface {
	SetCell(x, y int, ch rune, fg, bg Attribute)
	Size() (w, h int)
	Flip()
	Clear(bg Attribute)
}

type Style struct {
	FgColor, BgColor Attribute
	FgStyle, BgStyle Attribute
}

func (s1 Style) Merge(s2 Style) Style {
	if s1.FgColor == 0 {
		s1.FgColor = s2.FgColor
	}
	if s1.FgStyle == 0 {
		s1.FgStyle = s2.FgStyle
	}
	if s1.BgColor == 0 {
		s1.BgColor = s2.BgColor
	}
	if s1.BgStyle == 0 {
		s1.BgStyle = s2.BgStyle
	}
	return s1
}

type StyleAttr struct {
	Name  string
	Value Style
}

type InputState struct {
	mouseX      int
	mouseY      int
	mouseButton MouseButton

	keyPress Key
	chPress  rune
}

// Imterm is a simple immediate mode text ui library
// Items are placed in a simple top down, left to right pattern
// If an item's width is 0, it will resize to fill the remainder of the width
// An item's ID must be unique, and by default is the label passed to the item.  This can be overriden by calling .ID() first.
type Imterm struct {
	screen Screen

	curState  InputState
	nextState InputState

	mouseState MouseButton

	xPos int
	yPos int

	focusID string
	lastID  string
	nextID  string

	lastY int
	nextX int
	nextY int

	columnStack []struct{ x, y, maxy, w int }

	columnWidth int
	columnX     int
	columnY     int
	columnMaxY  int

	TermW int
	TermH int

	baseStyle  map[string]Style
	styleStack []StyleAttr

	widgetState map[string]interface{}

	lastBox Box
}

func (it *Imterm) ClearState() {
	for k := range it.widgetState {
		delete(it.widgetState, k)
	}
}

func (it *Imterm) getState(id string, def interface{}) interface{} {
	if state, ok := it.widgetState[id]; ok {
		return state
	}
	it.widgetState[id] = def
	return def
}

// set info about mouse actions
func (it *Imterm) Mouse(x, y int, button MouseButton) {
	if it.mouseState != button || button == MouseWheelUp || button == MouseWheelDown {
		it.nextState.mouseX = x
		it.nextState.mouseY = y
		it.nextState.mouseButton = button
		it.mouseState = button
	}
}

// Set info about keyboard presses.  Values are equivalent to termbox-go values
func (it *Imterm) Keyboard(key Key, ch rune) {
	it.nextState.keyPress = key
	it.nextState.chPress = ch
}

// Simple check what mouse button was clicked in a region
func (it *Imterm) CheckClick(x, y, w, h int) MouseButton {
	if it.curState.mouseButton != 0 {
		if it.curState.mouseX >= x && it.curState.mouseX < x+w &&
			it.curState.mouseY >= y && it.curState.mouseY < y+h {
			return it.curState.mouseButton
		}
	}
	return MouseNone
}

func (it *Imterm) GetClick(x, y, w, h int) (mx, my int, mb MouseButton) {
	if it.curState.mouseButton != 0 {
		if it.curState.mouseX >= x && it.curState.mouseX < x+w &&
			it.curState.mouseY >= y && it.curState.mouseY < y+h {
			return it.curState.mouseX - x, it.curState.mouseY - y, it.curState.mouseButton
		}
	}
	return 0, 0, MouseNone
}

// Was the last object focused?
func (it *Imterm) Focus() bool {
	return it.focusID == it.lastID
}

func (it *Imterm) setLast(id string) {
	it.lastID = id
}

// Set the focus to a specific ID
func (it *Imterm) SetFocus(id string) {
	it.focusID = id
}

// Override the next object to have the given ID
func (it *Imterm) ID(id string) *Imterm {
	it.nextID = id
	return it
}

func (it *Imterm) getID(id string) (ret string) {
	if it.nextID != "" {
		ret, it.nextID = it.nextID, ""
		return
	}
	ret = id
	return
}

type Box struct {
	x, y, w, h int
}

func (it *Imterm) getBox(w, h int) (b Box) {
	if w <= 0 {
		w = ((it.columnX + it.columnWidth) - it.xPos) + w
	}
	if h <= 0 {
		h = (it.TermH - it.yPos) + h
	}
	b = Box{it.xPos, it.yPos, w, h}
	it.lastBox = b
	it.nextX = it.xPos + w
	it.lastY = it.yPos
	it.xPos, it.yPos = it.columnX, it.yPos+h
	if it.yPos < it.nextY {
		it.yPos = it.nextY
	}
	if it.columnMaxY < it.yPos {
		it.columnMaxY = it.yPos
	}
	return
}

func (it *Imterm) StartColumns(w int) {
	it.columnStack = append(it.columnStack, struct{ x, y, maxy, w int }{it.columnX, it.columnY, it.columnMaxY, it.columnWidth})
	it.columnY = it.yPos
	it.columnWidth = w
}

func (it *Imterm) NextColumn(w int) {
	it.columnX = it.columnX + it.columnWidth
	if w == 0 {
		w = it.columnStack[len(it.columnStack)-1].w - it.columnX
	}
	it.columnWidth = w
	it.yPos = it.columnY
	it.xPos = it.columnX
	it.nextY = it.yPos
}

func (it *Imterm) FinishColumns() {
	it.yPos = it.columnMaxY
	it.columnX, it.columnY, it.columnMaxY, it.columnWidth = it.columnStack[len(it.columnStack)-1].x, it.columnStack[len(it.columnStack)-1].y, it.columnStack[len(it.columnStack)-1].maxy, it.columnStack[len(it.columnStack)-1].w
	it.columnStack = it.columnStack[:len(it.columnStack)-1]
	it.xPos = it.columnX
}

func (it *Imterm) GetBaseStyle(name string) Style {
	val := Style{}
	if it.Focus() {
		nval, ok := it.baseStyle[name+":focus"]
		if ok {
			val = val.Merge(nval)
		}
	}
	nval, ok := it.baseStyle[name]
	if ok {
		val = val.Merge(nval)
	}
	val = val.Merge(nval)
	split := strings.SplitAfter(name, ".")
	splitlen := 0
	for _, s := range split {
		splitlen += len(s)
		partname := name[splitlen:]
		if it.Focus() {
			nval, ok = it.baseStyle[partname+":focus"]
			if ok {
				val = val.Merge(nval)
			}
		}
		nval, ok := it.baseStyle[partname]
		if ok {
			val = val.Merge(nval)
		}

	}
	return val
}

type CalcedStyle struct {
	Fg, Bg Attribute
}

func (it *Imterm) GetStyle(name string) CalcedStyle {
	val := it.GetBaseStyle(name)
	for _, s := range it.styleStack {
		if s.Name == name || strings.HasSuffix(s.Name, "."+name) {
			val = s.Value.Merge(val)
		}
		if it.Focus() && (s.Name == name+":focus" || strings.HasSuffix(s.Name, "."+name+":focus")) {
			val = s.Value.Merge(val)
		}
	}
	return CalcedStyle{
		Fg: val.FgColor | val.FgStyle,
		Bg: val.BgColor | val.BgStyle,
	}
}

func (it *Imterm) PushStyle(name string, style Style) {
	it.styleStack = append(it.styleStack, StyleAttr{name, style})
}

func (it *Imterm) PopStyles(num int) {
	it.styleStack = it.styleStack[:len(it.styleStack)-num]
}

func New(screen Screen) (*Imterm, error) {
	it := &Imterm{
		screen: screen,
		baseStyle: map[string]Style{
			"border:focus":       Style{FgStyle: AttrBold},
			"border.label:focus": Style{FgStyle: AttrBold},
			"active.border":      Style{FgColor: ColorGreen},
			"gauge.bar.on":       Style{BgColor: ColorRed},
		},
		widgetState: map[string]interface{}{},
	}
	it.TermW, it.TermH = screen.Size()
	return it, nil
}

// Start a frame, this must be called before drawing any objects to the screen
func (it *Imterm) Start() {
	it.TermW, it.TermH = it.screen.Size()
	it.xPos, it.yPos = 0, 0
	it.nextX, it.nextY, it.lastY = 0, 0, 0
	it.screen.Clear(it.GetStyle("").Bg)
	it.curState = it.nextState
	it.nextState = InputState{}

	it.columnX, it.columnY = 0, 0
	it.columnMaxY = 0
	it.columnWidth = it.TermW
	it.columnStack = it.columnStack[:0]
}

func (it *Imterm) hLine(x, y int, w int, s CalcedStyle) {
	for i := 0; i <= w; i++ {
		it.screen.SetCell(x+i, y, '─', s.Fg, s.Bg)
	}
}

func (it *Imterm) vLine(x, y int, h int, s CalcedStyle) {
	for i := 0; i <= h; i++ {
		it.screen.SetCell(x, y+i, '│', s.Fg, s.Bg)
	}
}

func (it *Imterm) frame(b Box, label string, class string) {
	s := it.GetStyle(class)
	x, y, w, h := b.x, b.y, b.w, b.h

	it.hLine(x+1, y, w-3, s)
	it.hLine(x+1, y+h-1, w-3, s)
	it.vLine(x, y+1, h-3, s)
	it.vLine(x+w-1, y+1, h-3, s)
	it.screen.SetCell(x, y, '┌', s.Fg, s.Bg)
	it.screen.SetCell(x+w-1, y, '┐', s.Fg, s.Bg)
	it.screen.SetCell(x, y+h-1, '└', s.Fg, s.Bg)
	it.screen.SetCell(x+w-1, y+h-1, '┘', s.Fg, s.Bg)
	if label != "" {
		it.screen.SetCell(x+1, y, '◄', s.Fg, s.Bg)
		ls := it.GetStyle(class + ".label")
		var i int
		for i = 0; i < w-4; i++ {
			if i >= len(label) {
				break
			}
			it.screen.SetCell(x+i+2, y, rune(label[i]), ls.Fg, ls.Bg)
		}
		it.screen.SetCell(x+i+2, y, '►', s.Fg, s.Bg)
	}
}

type textState struct {
	scroll int
}

func (it *Imterm) GetLast() (x, y, w, h int) {
	return it.lastBox.x, it.lastBox.y, it.lastBox.w, it.lastBox.h
}

// Place a text label.  Not editable
func (it *Imterm) Text(w, h int, label string, text string) {
	id := it.getID(label)
	it.setLast(id)
	b := it.getBox(w, h)

	it.frame(b, label, "text.border")

	s := it.GetStyle("text.text")

	state := it.getState(id, &textState{}).(*textState)

	it.screen.SetCell(b.x+b.w-1, b.y+1, '▲', s.Fg, s.Bg)
	if state.scroll > 0 {
		if it.CheckClick(b.x+b.w-1, b.y+1, 1, 1) == MouseLeft {
			state.scroll--
		}
		if it.CheckClick(b.x, b.y, b.w, b.h) == MouseWheelUp {
			state.scroll--
		}
	}

	wrapped := wordwrap.WrapString(text, uint(b.w-2))
	cx, cy := 0, 0

	more := false
	for _, r := range wrapped {
		if r == '\n' {
			cx, cy = 0, cy+1
			if cy-state.scroll >= b.h-2 {
				more = true
				break
			}
		} else {
			if cy-state.scroll >= 0 {
				it.screen.SetCell(b.x+1+cx, b.y+1+cy-state.scroll, r, s.Fg, s.Bg)
			}
			cx++
		}
	}
	it.screen.SetCell(b.x+b.w-1, b.y+b.h-2, '▼', s.Fg, s.Bg)
	if more {
		if it.CheckClick(b.x+b.w-1, b.y+b.h-2, 1, 1) == MouseLeft {
			state.scroll++
		}
		if it.CheckClick(b.x, b.y, b.w, b.h) == MouseWheelDown {
			state.scroll++
		}
	}
}

type Cell struct {
	Char   rune
	Fg, Bg Attribute
}

type bufferState struct {
	xscroll, yscroll int
}

// Place a buffer.  Not editable, but responds to click events.  Expects buffer to be uniform in size for all rows.
func (it *Imterm) Buffer(w, h int, label string, buffer [][]Cell) (mx, my int, mb MouseButton) {
	id := it.getID(label)
	it.setLast(id)
	b := it.getBox(w, h)

	it.frame(b, label, "text.border")

	s := it.GetStyle("text.text")

	state := it.getState(id, &bufferState{}).(*bufferState)
	it.screen.SetCell(b.x+b.w-1, b.y+1, '▲', s.Fg, s.Bg)
	if state.yscroll > 0 {
		if it.CheckClick(b.x+b.w-1, b.y+1, 1, 1) == MouseLeft {
			state.yscroll--
		}
		if it.CheckClick(b.x, b.y, b.w, b.h) == MouseWheelUp {
			state.yscroll--
		}
	}
	it.screen.SetCell(b.x+b.w-1, b.y+b.h-2, '▼', s.Fg, s.Bg)
	if state.yscroll+b.h-2 < len(buffer) {
		if it.CheckClick(b.x+b.w-1, b.y+b.h-2, 1, 1) == MouseLeft {
			state.yscroll++
		}
		if it.CheckClick(b.x, b.y, b.w, b.h) == MouseWheelDown {
			state.yscroll++
		}
	} else if state.yscroll+b.h-2 > len(buffer) {
		state.yscroll = len(buffer) - b.h - 2
		if state.yscroll < 0 {
			state.yscroll = 0
		}
	}

	it.screen.SetCell(b.x+1, b.y+b.h-1, '◄', s.Fg, s.Bg)
	if state.xscroll > 0 {
		if it.CheckClick(b.x+1, b.y+b.h-1, 1, 1) == MouseLeft {
			state.xscroll--
		}
	}
	it.screen.SetCell(b.x+b.w-2, b.y+b.h-1, '►', s.Fg, s.Bg)
	if state.xscroll+b.w-2 < len(buffer[0]) {
		if it.CheckClick(b.x+b.w-2, b.y+b.h-1, 1, 1) == MouseLeft {
			state.xscroll++
		}
	} else if state.xscroll+b.w-2 > len(buffer[0]) {
		state.xscroll = len(buffer[0]) - b.w - 2
		if state.xscroll < 0 {
			state.xscroll = 0
		}
	}

	for cy := 0; cy < b.h-2; cy++ {
		if cy+state.yscroll >= len(buffer) {
			break
		}
		row := buffer[cy+state.yscroll]
		for cx := 0; cx < b.w-2; cx++ {
			if cx+state.xscroll >= len(row) {
				break
			}
			cell := row[cx+state.xscroll]
			it.screen.SetCell(b.x+1+cx, b.y+1+cy, cell.Char, cell.Fg, cell.Bg)
		}
	}
	return it.GetClick(b.x+1, b.y+1, b.w-2, b.h-2)
}

type inputState struct {
	cPos int
}

// Place an editable text area
func (it *Imterm) Input(w, h int, label string, text string) string {
	id := it.getID(label)
	it.setLast(id)
	b := it.getBox(w, h)
	x, y, w, h := b.x, b.y, b.w, b.h

	state := it.getState(id, &inputState{cPos: -1}).(*inputState)
	mx, my := -1, -1

	if state.cPos == -1 {
		state.cPos = len(text)
	}

	if it.CheckClick(x, y, w, h) == MouseLeft {
		mx = it.curState.mouseX - (x + 1)
		my = it.curState.mouseY - (y + 1)
		if mx < 0 {
			mx = 0
		} else if mx > w-2 {
			mx = w - 2
		}
		if my < 0 {
			my = 0
		} else if my > h-2 {
			my = h - 2
		}
		it.focusID = id
	}

	if it.Focus() {
		if it.curState.chPress != 0 {
			text = text[:state.cPos] + string(it.curState.chPress) + text[state.cPos:]
			state.cPos++
		} else if it.curState.keyPress != 0 {
			switch it.curState.keyPress {
			case KeyBackspace, KeyBackspace2:
				if state.cPos > 0 {
					text = text[:state.cPos-1] + text[state.cPos:]
					state.cPos--
				}
			case KeyDelete:
				text = text[:state.cPos] + text[state.cPos+1:]
			case KeySpace:
				text = text[:state.cPos] + " " + text[state.cPos:]
				state.cPos++
			case KeyEnter:
				text = text[:state.cPos] + "\n" + text[state.cPos:]
				state.cPos++
			case KeyArrowLeft:
				if state.cPos > 0 {
					state.cPos--
				}
			case KeyArrowRight:
				if state.cPos < len(text) {
					state.cPos++
				}
			default:

			}
		}
	}
	if state.cPos < 0 {
		state.cPos = 0
	}
	if state.cPos > len(text) {
		state.cPos = len(text)
	}

	it.frame(b, label, "input.border")

	s := it.GetStyle("input.text")

	//wrapped := wordwrap.WrapString(text, uint(w-2))
	cx, cy := 0, 0
	showcursor := false
	if it.Focus() {
		showcursor = true
	}
	cursor := false
	nextspace := 0
	for i, r := range text {
		if mx >= 0 && cx == mx && cy == my {
			state.cPos = i
			mx = -1
		}
		if r == '\n' {
			if i == state.cPos && showcursor {
				it.screen.SetCell(cx+x+1, cy+y+1, ' ', s.Fg|AttrUnderline, s.Bg|AttrUnderline)
				cursor = true
			}

			cx, cy = 0, cy+1
			if mx >= 0 && cy > my {
				state.cPos = i
				mx = -1
			}
			if cy >= h-2 {
				break
			}
		} else {
			if i >= nextspace {
				nextspace = strings.IndexAny(text[i:], "\n\t ")
				if nextspace == -1 {
					nextspace = len(text[i:])
				}
				if nextspace > (w-2)-cx {
					cx, cy = 0, cy+1
					if mx >= 0 && cy > my {
						state.cPos = i
						mx = -1
					}
					if cy >= h-2 {
						break
					}
				}
				nextspace += i
			}
			if cx >= w-2 {
				cx, cy = 0, cy+1
				if cy >= h-2 {
					break
				}
			}
			if i == state.cPos && showcursor {
				it.screen.SetCell(cx+x+1, cy+y+1, r, s.Fg|AttrUnderline, s.Bg|AttrUnderline)
				cursor = true
			} else {
				it.screen.SetCell(cx+x+1, cy+y+1, r, s.Fg, s.Bg)
			}
			cx++
		}
	}
	if mx >= 0 {
		state.cPos = len(text)
		mx = -1
	}
	if !cursor && cy < h-2 && showcursor {
		it.screen.SetCell(cx+x+1, cy+y+1, ' ', s.Fg|AttrUnderline, s.Bg|AttrUnderline)
	}
	return text
}

// Place a clickable button
func (it *Imterm) Button(w, h int, label string) bool {
	id := it.getID(label)
	it.setLast(id)
	b := it.getBox(w, h)
	x, y, w, h := b.x, b.y, b.w, b.h

	click := false

	if it.CheckClick(x, y, w, h) == MouseLeft {
		it.SetFocus(id)
		click = true
	}

	it.frame(b, "", "button.border")
	s := it.GetStyle("button.text")
	wrapped := wordwrap.WrapString(label, uint(w-2))
	cx, cy := x+1, y+1

	for _, r := range wrapped {
		if r == '\n' {
			cx, cy = x+1, cy+1
			if cy >= y+h {
				break
			}
		} else {
			it.screen.SetCell(cx, cy, r, s.Fg, s.Bg)
			cx++
		}
	}

	return click
}

// Place a toggleable button
func (it *Imterm) Toggle(w, h int, label string, state bool) bool {
	id := it.getID(label)
	it.setLast(id)
	b := it.getBox(w, h)
	x, y, w, h := b.x, b.y, b.w, b.h

	click := false

	if it.CheckClick(x, y, w, h) == MouseLeft {
		it.SetFocus(id)
		click = true
	}

	if click {
		state = !state
	}

	class := "toggle"
	if state {
		class += ".active"
	}

	it.frame(b, "", class+".border")
	s := it.GetStyle(class + ".text")
	wrapped := wordwrap.WrapString(label, uint(w-2))
	cx, cy := x+1, y+1

	for _, r := range wrapped {
		if r == '\n' {
			cx, cy = x+1, cy+1
			if cy >= y+h {
				break
			}
		} else {
			it.screen.SetCell(cx, cy, r, s.Fg, s.Bg)
			cx++
		}
	}

	return state
}

// Place a gauge, percent is a float from 0-1
func (it *Imterm) Gauge(w, h int, label string, percent float32, overlay string) {
	id := it.getID(label)
	it.setLast(id)
	b := it.getBox(w, h)
	x, y, w, h := b.x, b.y, b.w, b.h

	overlaywidth := len(overlay)

	it.frame(b, label, "gauge.border")
	s := it.GetStyle("gauge.bar.on")
	wasactive := true
	for cx := 0; cx < w-2; cx++ {
		if wasactive && (float32(cx)/float32(w-1)) >= percent {
			s = it.GetStyle("gauge.bar.off")
		}
		overlayPrinted := false
		for cy := 0; cy < h-2; cy++ {
			if !overlayPrinted && (cy+1) > ((h-2)/2) && cx >= ((w-2)/2-(overlaywidth/2)) && cx < ((w-2)/2-(overlaywidth/2))+overlaywidth {
				it.screen.SetCell(cx+x+1, cy+y+1, rune(overlay[cx-((w-2)/2-(overlaywidth/2))]), s.Fg, s.Bg)
				overlayPrinted = true
			} else {
				it.screen.SetCell(cx+x+1, cy+y+1, ' ', s.Fg, s.Bg)
			}
		}
	}
}

type listState struct {
	scroll int
}

// Place a list area, user can scroll if there are too many items
func (it *Imterm) List(w, h int, label string, contents []string) {
	id := it.getID(label)
	it.setLast(id)
	b := it.getBox(w, h)
	x, y, w, h := b.x, b.y, b.w, b.h

	it.frame(b, label, "list.border")

	s := it.GetStyle("list.items")

	state := it.getState(label, &listState{}).(*listState)

	cy := 0

	it.screen.SetCell(x+w-1, y+1, '▲', s.Fg, s.Bg)
	if state.scroll > 0 {
		if it.CheckClick(x+w-1, y+1, 1, 1) == MouseLeft {
			state.scroll--
		}
	}
	it.screen.SetCell(x+w-1, y+h-2, '▼', s.Fg, s.Bg)
	if state.scroll < len(contents)-(h-2) {
		if it.CheckClick(x+w-1, y+h-2, 1, 1) == MouseLeft {
			state.scroll++
		}
	}

	for cy = 0; cy < h-2; cy++ {
		cx := 0
		if cy+state.scroll >= len(contents) {
			break
		}
		for _, ch := range contents[cy+state.scroll] {
			if cx > w-2 {
				break
			}
			it.screen.SetCell(cx+x+1, cy+y+1, ch, s.Fg, s.Bg)
			cx++
		}
	}
}

type selectableListState struct {
	scroll int
}

// Place a selectable list.  User can scroll, and returns a slice of ints of which indexes are selected.
func (it *Imterm) SelectableList(w, h int, label string, contents []string, selected []int) []int {
	id := it.getID(label)
	it.setLast(id)
	b := it.getBox(w, h)
	x, y, w, h := b.x, b.y, b.w, b.h

	if it.CheckClick(x, y, w, h) == MouseLeft {
		it.SetFocus(id)
	}

	it.frame(b, label, "list.border")

	s := it.GetStyle("list.items")

	state := it.getState(label, &selectableListState{}).(*selectableListState)

	cy := 0

	it.screen.SetCell(x+w-1, y+1, '▲', s.Fg, s.Bg)
	if state.scroll > 0 {
		if it.CheckClick(x+w-1, y+1, 1, 1) == MouseLeft {
			state.scroll--
		}
	}
	it.screen.SetCell(x+w-1, y+h-2, '▼', s.Fg, s.Bg)
	if state.scroll < len(contents)-(h-2) {
		if it.CheckClick(x+w-1, y+h-2, 1, 1) == MouseLeft {
			state.scroll++
		}
	}

	for cy = 0; cy < h-2; cy++ {
		cx := 0
		if cy+state.scroll >= len(contents) {
			break
		}
		iselected := false
		selindex := 0
		for i, v := range selected {
			if v == cy+state.scroll {
				iselected = true
				selindex = i
				break
			}
		}
		if it.CheckClick(x+1, y+cy+1, w-2, 1) == MouseLeft {
			if iselected {
				selected = append(selected[:selindex], selected[selindex+1:]...)
			} else {
				selected = append(selected, cy+state.scroll)
			}
			iselected = !iselected
		}
		for _, ch := range contents[cy+state.scroll] {
			if cx >= w-2 {
				break
			}
			if !iselected {
				it.screen.SetCell(cx+x+1, cy+y+1, ch, s.Fg, s.Bg)
			} else {
				it.screen.SetCell(cx+x+1, cy+y+1, ch, s.Fg|AttrReverse, s.Bg|AttrReverse)
			}
			cx++
		}
		if iselected {
			for ; cx < w-2; cx++ {
				it.screen.SetCell(cx+x+1, cy+y+1, ' ', s.Fg|AttrReverse, s.Bg|AttrReverse)
			}
		}
	}
	return selected
}

// Positions the next item to the right of the prior item
func (it *Imterm) SameLine() {
	if it.nextY < it.yPos {
		it.nextY = it.yPos
	}
	it.yPos = it.lastY
	it.xPos = it.nextX
}

// Finishes and renders the frame
func (it *Imterm) Finish() {
	it.screen.Flip()
}
