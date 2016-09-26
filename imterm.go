package imterm

import (
	"log"
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

	TermW int
	TermH int

	baseStyle  map[string]Style
	styleStack []StyleAttr

	widgetState map[string]interface{}
}

func (it *Imterm) getState(id string, def interface{}) interface{} {
	if state, ok := it.widgetState[id]; ok {
		return state
	}
	it.widgetState[id] = def
	return def
}

func (it *Imterm) Mouse(x, y int, button MouseButton) {
	if it.mouseState != button {
		it.nextState.mouseX = x
		it.nextState.mouseY = y
		it.nextState.mouseButton = button
		it.mouseState = button
	}
}

func (it *Imterm) Keyboard(key Key, ch rune) {
	it.nextState.keyPress = key
	it.nextState.chPress = ch
}

func (it *Imterm) CheckClick(x, y, w, h int) MouseButton {
	if it.curState.mouseButton != 0 {
		if it.curState.mouseX >= x && it.curState.mouseX < x+w &&
			it.curState.mouseY >= y && it.curState.mouseY < y+h {
			return it.curState.mouseButton
		}
	}
	return 0
}

func (it *Imterm) Focus() bool {
	return it.focusID == it.lastID
}

func (it *Imterm) setLast(id string) {
	it.lastID = id
}

func (it *Imterm) SetFocus(id string) {
	it.focusID = id
}

func (it *Imterm) ID(id string) {
	it.nextID = id
}

func (it *Imterm) getID(id string) (ret string) {
	if it.nextID != "" {
		ret, it.nextID = it.nextID, ""
		return
	}
	ret = id
	return
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
			val = val.Merge(s.Value)
		}
		if it.Focus() && (s.Name == name+":focus" || strings.HasSuffix(s.Name, "."+name+":focus")) {
			val = val.Merge(s.Value)
		}
	}
	return CalcedStyle{
		Fg: val.FgColor | val.FgStyle,
		Bg: val.BgColor | val.BgStyle,
	}
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

func (it *Imterm) Start() {
	it.TermW, it.TermH = it.screen.Size()
	it.xPos, it.yPos = 0, 0
	it.nextX, it.nextY, it.lastY = 0, 0, 0
	it.screen.Clear(it.GetStyle("").Bg)
	it.curState = it.nextState
	it.nextState = InputState{}
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

func (it *Imterm) frame(w, h int, label string, class string) {
	s := it.GetStyle(class)
	if w == 0 {
		w = it.TermW - it.xPos
	}
	x, y := it.xPos, it.yPos

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
	it.lastY = y
	it.nextX = x + w
	it.xPos = 0
	it.yPos = y + h
	if it.yPos < it.nextY {
		it.yPos = it.nextY
	}
}

func (it *Imterm) Text(w, h int, text string, label string) {
	it.getID(label)
	it.setLast(label)
	if w == 0 {
		w = it.TermW - it.xPos
	}
	x, y := it.xPos, it.yPos

	it.frame(w, h, label, "text.border")

	s := it.GetStyle("text.text")

	wrapped := wordwrap.WrapString(text, uint(w-2))
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
}

type inputState struct {
	cPos int
}

func (it *Imterm) Input(w, h int, text string, label string) string {
	id := it.getID(label)
	it.setLast(label)
	if w == 0 {
		w = it.TermW - it.xPos
	}
	x, y := it.xPos, it.yPos

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
			log.Printf("KeyPress: %x", it.curState.keyPress)
		}
	}
	if state.cPos < 0 {
		state.cPos = 0
	}
	if state.cPos > len(text) {
		state.cPos = len(text)
	}

	it.frame(w, h, label, "input.border")

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
		if r == '\n' {
			if i == state.cPos && showcursor {
				it.screen.SetCell(cx+x+1, cy+y+1, ' ', s.Fg|AttrUnderline, s.Bg|AttrUnderline)
				cursor = true
			}

			cx, cy = 0, cy+1
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
	if !cursor && cy < h-2 && showcursor {
		it.screen.SetCell(cx+x+1, cy+y+1, ' ', s.Fg|AttrUnderline, s.Bg|AttrUnderline)
	}
	return text
}

func (it *Imterm) Button(w, h int, label string) bool {
	id := it.getID(label)
	it.setLast(label)
	if w == 0 {
		w = it.TermW - it.xPos
	}
	x, y := it.xPos, it.yPos

	click := false

	if it.CheckClick(x, y, w, h) == MouseLeft {
		it.SetFocus(id)
		click = true
	}

	it.frame(w, h, "", "button.border")
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

type toggleState struct {
	status bool
}

func (it *Imterm) Toggle(w, h int, label string) bool {
	id := it.getID(label)
	it.setLast(label)
	if w == 0 {
		w = it.TermW - it.xPos
	}
	x, y := it.xPos, it.yPos

	click := false

	if it.CheckClick(x, y, w, h) == MouseLeft {
		it.SetFocus(id)
		click = true
	}

	state := it.getState(id, &toggleState{}).(*toggleState)

	if click {
		state.status = !state.status
	}

	class := "toggle"
	if state.status {
		class += ".active"
	}

	it.frame(w, h, "", class+".border")
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

	return state.status
}

func (it *Imterm) Gauge(w, h int, label string, percent float32, overlay string) {
	it.getID(label)
	it.setLast(label)
	if w == 0 {
		w = it.TermW - it.xPos
	}
	x, y := it.xPos, it.yPos

	overlaywidth := len(overlay)

	it.frame(w, h, label, "gauge.border")
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

func (it *Imterm) List(w, h int, label string, contents []string) {
	it.getID(label)
	it.setLast(label)
	if w == 0 {
		w = it.TermW - it.xPos
	}
	x, y := it.xPos, it.yPos

	it.frame(w, h, label, "list.border")

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
	scroll   int
	selected []int
}

func (it *Imterm) SelectableList(w, h int, label string, contents []string) []int {
	id := it.getID(label)
	it.setLast(label)
	if w == 0 {
		w = it.TermW - it.xPos
	}
	x, y := it.xPos, it.yPos

	if it.CheckClick(x, y, w, h) == MouseLeft {
		it.SetFocus(id)
	}

	it.frame(w, h, label, "list.border")

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
		selected := false
		selindex := 0
		for i, v := range state.selected {
			if v == cy+state.scroll {
				selected = true
				selindex = i
				break
			}
		}
		if it.CheckClick(x+1, y+cy+1, w-2, 1) == MouseLeft {
			if selected {
				state.selected = append(state.selected[:selindex], state.selected[selindex+1:]...)
			} else {
				state.selected = append(state.selected, cy+state.scroll)
			}
			selected = !selected
		}
		for _, ch := range contents[cy+state.scroll] {
			if cx > w-2 {
				break
			}
			if !selected {
				it.screen.SetCell(cx+x+1, cy+y+1, ch, s.Fg, s.Bg)
			} else {
				it.screen.SetCell(cx+x+1, cy+y+1, ch, s.Fg|AttrReverse, s.Bg|AttrReverse)
			}
			cx++
		}
		if selected {
			for ; cx < w-2; cx++ {
				it.screen.SetCell(cx+x+1, cy+y+1, ' ', s.Fg|AttrReverse, s.Bg|AttrReverse)
			}
		}
	}
	return state.selected
}

func (it *Imterm) SameLine() {
	if it.nextY < it.yPos {
		it.nextY = it.yPos
	}
	it.yPos = it.lastY
	it.xPos = it.nextX
}

func (it *Imterm) Finish() {
	it.screen.Flip()
}
