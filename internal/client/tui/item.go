package tui

import (
	"time"

	"github.com/bep/debounce"
	"github.com/gcla/gowid"
	"github.com/gcla/gowid/gwutil"
	"github.com/gcla/gowid/widgets/columns"
	"github.com/gcla/gowid/widgets/edit"
	"github.com/gcla/gowid/widgets/selectable"
	"github.com/gcla/gowid/widgets/styled"
	"github.com/gcla/gowid/widgets/text"
	"github.com/gcla/gowid/widgets/vscroll"
	"github.com/gdamore/tcell/v2"
	"github.com/mdouchement/standardfile/pkg/libsf"
)

// An Item is the graphical representation of an libsf.Item.
type Item struct {
	ID                 string
	presentation       gowid.IWidget
	abstraction        *libsf.Item
	editorPresentation *ItemEditor
}

// NewItem returns a new Item.
func NewItem(item *libsf.Item, sync func(item *libsf.Item) *time.Time) *Item {
	editor := edit.New(edit.Options{Text: item.Note.Text})
	debounced := debounce.New(500 * time.Millisecond)
	editor.OnTextSet(gowid.WidgetCallback{Name: "cb", WidgetChangedFunction: func(app gowid.IApp, iw gowid.IWidget) {
		debounced(func() {
			item.Note.Text = editor.Text()
			item.UpdatedAt = sync(item)
		})
	}})

	return &Item{
		ID: item.ID,
		presentation: selectable.New(
			styled.NewExt(
				text.New(item.Note.Title),
				gowid.MakePaletteRef("normal"), gowid.MakePaletteRef("focused"),
			),
		),
		editorPresentation: newItemEditor(editor),
		abstraction:        item,
	}
}

// Title returns the name of the editor.
func (w *Item) Title() string {
	return w.abstraction.Note.Title
}

// Editor returns the ItemContent of the Item.
func (w *Item) Editor() *ItemEditor {
	return w.editorPresentation
}

////////////////////
//                //
// Delegates      //
//                //
////////////////////

// Render implements gowid.IWidget
func (w *Item) Render(size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) gowid.ICanvas {
	return w.presentation.Render(size, focus, app)
}

// RenderSize implements gowid.IWidget
func (w *Item) RenderSize(size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) gowid.IRenderBox {
	return w.presentation.RenderSize(size, focus, app)
}

// UserInput implements gowid.IWidget
func (w *Item) UserInput(ev any, size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) bool {
	return w.presentation.UserInput(ev, size, focus, app)
}

// Selectable implements gowid.IWidget
func (w *Item) Selectable() bool {
	return w.presentation.Selectable()
}

//
//
//
//
//
//
//
//
//
//
//
//

// An ItemEditor is the graphical representation of editable libsf.Item text.
type ItemEditor struct {
	*columns.Widget
	e        *edit.Widget
	sb       *vscroll.Widget
	goUpDown int // positive means down
	pgUpDown int // positive means down
}

func newItemEditor(e *edit.Widget) *ItemEditor {
	sb := vscroll.NewExt(vscroll.VerticalScrollbarUnicodeRunes)
	ie := &ItemEditor{
		Widget: columns.New([]gowid.IContainerWidget{
			&gowid.ContainerWidget{IWidget: e, D: gowid.RenderWithWeight{W: 1}},
			&gowid.ContainerWidget{IWidget: sb, D: gowid.RenderWithUnits{U: 1}},
		}),
		e:        e,
		sb:       sb,
		goUpDown: 0,
		pgUpDown: 0,
	}
	sb.OnClickAbove(gowid.WidgetCallback{Name: "cb", WidgetChangedFunction: ie.clickUp})
	sb.OnClickBelow(gowid.WidgetCallback{Name: "cb", WidgetChangedFunction: ie.clickDown})
	sb.OnClickUpArrow(gowid.WidgetCallback{Name: "cb", WidgetChangedFunction: ie.clickUpArrow})
	sb.OnClickDownArrow(gowid.WidgetCallback{Name: "cb", WidgetChangedFunction: ie.clickDownArrow})
	return ie
}

func (w *ItemEditor) clickUp(app gowid.IApp, iw gowid.IWidget) {
	w.pgUpDown--
}

func (w *ItemEditor) clickDown(app gowid.IApp, iw gowid.IWidget) {
	w.pgUpDown++
}

func (w *ItemEditor) clickUpArrow(app gowid.IApp, iw gowid.IWidget) {
	w.goUpDown--
}

func (w *ItemEditor) clickDownArrow(app gowid.IApp, iw gowid.IWidget) {
	w.goUpDown++
}

// UserInput implements gowid.IWidget
func (w *ItemEditor) UserInput(ev any, size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) bool {
	box, _ := size.(gowid.IRenderBox)
	w.sb.Top, w.sb.Middle, w.sb.Bottom = w.e.CalculateTopMiddleBottom(gowid.MakeRenderBox(box.BoxColumns()-1, box.BoxRows()))

	// Remap events
	if k, ok := ev.(*tcell.EventKey); ok {
		switch k.Key() {
		case tcell.KeyHome:
			ev = tcell.NewEventKey(tcell.KeyCtrlA, ' ', tcell.ModNone) // Start of line defined by edit widget
		case tcell.KeyEnd:
			ev = tcell.NewEventKey(tcell.KeyCtrlE, ' ', tcell.ModNone) // End of line defined by edit widget
		}
	}

	handled := w.Widget.UserInput(ev, size, focus, app)
	if handled {
		w.Widget.SetFocus(app, 0)
	}

	return handled
}

// Render implements gowid.IWidget
func (w *ItemEditor) Render(size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) gowid.ICanvas {
	box, _ := size.(gowid.IRenderBox)
	ecols := box.BoxColumns() - 1
	ebox := gowid.MakeRenderBox(ecols, box.BoxRows())
	if w.goUpDown != 0 || w.pgUpDown != 0 {
		w.e.SetLinesFromTop(gwutil.Max(0, w.e.LinesFromTop()+w.goUpDown+(w.pgUpDown*box.BoxRows())), app)
		txt := w.e.MakeText()
		layout := text.MakeTextLayout(txt.Content(), ecols, txt.Wrap(), gowid.HAlignLeft{})
		_, y := text.GetCoordsFromCursorPos(w.e.CursorPos(), ecols, layout, w.e)
		if y < w.e.LinesFromTop() {
			for i := y; i < w.e.LinesFromTop(); i++ {
				w.e.DownLines(ebox, false, app)
			}
		} else if y >= w.e.LinesFromTop()+box.BoxRows() {
			for i := w.e.LinesFromTop() + box.BoxRows(); i <= y; i++ {
				w.e.UpLines(ebox, false, app)
			}
		}
	}
	w.goUpDown = 0
	w.pgUpDown = 0
	w.sb.Top, w.sb.Middle, w.sb.Bottom = w.e.CalculateTopMiddleBottom(ebox)

	return w.Widget.Render(size, focus, app)
}
