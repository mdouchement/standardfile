package tui

import (
	"fmt"
	"sort"

	"github.com/gcla/gowid"
	"github.com/gcla/gowid/widgets/list"
	"github.com/gdamore/tcell/v2"
)

// A NoteList is a list of Items to interract with.
// It implements gowid.IWidget by delegating to its presentation.
type NoteList struct {
	ui           *TUI
	presentation list.IWidget
	abstraction  *noteListAbstraction
}

// NewNoteList returns a new NoteList.
func NewNoteList(ui *TUI) *NoteList {
	abs := newNoteListAbstraction()

	return &NoteList{
		ui:           ui,
		presentation: list.New(abs),
		abstraction:  abs,
	}
}

// Register registers an item to this list.
func (w *NoteList) Register(i *Item) {
	n := w.abstraction.Add(i)
	if n == 1 {
		w.hackToDisplayFirstNote()
	}
}

// Sort orders items by the given field.
func (w *NoteList) Sort(field string) {
	if !w.abstraction.Sort(field) {
		msg := "No sort as been applied"
		if len(field) > 0 {
			msg = fmt.Sprintf("Failed to sort on %s, fallback on the item's name", field)
		}

		w.ui.DisplayStatus(msg)
		return
	}

	if w.abstraction.Length() > 0 {
		w.hackToDisplayFirstNote()
	}
}

// Hack to display first note content.
func (w *NoteList) hackToDisplayFirstNote() {
	w.ui.editor.SetTitle(w.abstraction.ItemAt(0).Title(), w.ui.App)
	w.ui.editor.SetSubWidget(w.abstraction.ItemAt(0).Editor(), w.ui.App)
}

////////////////////
//                //
// Delegates      //
//                //
////////////////////

// Render implements gowid.IWidget
func (w *NoteList) Render(size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) gowid.ICanvas {
	return w.presentation.Render(size, focus, app)
}

// RenderSize implements gowid.IWidget
func (w *NoteList) RenderSize(size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) gowid.IRenderBox {
	return w.presentation.RenderSize(size, focus, app)
}

// UserInput implements gowid.IWidget
func (w *NoteList) UserInput(ev any, size gowid.IRenderSize, focus gowid.Selector, app gowid.IApp) bool {
	ok := w.presentation.UserInput(ev, size, focus, app)

	if evm, ok := ev.(*tcell.EventMouse); !ok || evm.Buttons() != tcell.ButtonNone {
		// Avoid next action on mouse hover event
		if item, ok := w.abstraction.At(w.abstraction.Focus()).(*Item); ok {
			// Set editor name
			w.ui.editor.SetTitle(item.Title(), app)
			// Display the text to edit
			w.ui.editor.SetSubWidget(item.Editor(), app)
		}
	}
	return ok
}

// Selectable implements gowid.IWidget
func (w *NoteList) Selectable() bool {
	return w.presentation.Selectable()
}

////////////////////
//                //
// Abstraction    //
//                //
////////////////////

// A noteListAbstraction is a list of Items to interract with.
// It implements list.IWalker interface.
type noteListAbstraction struct {
	widgets   []*Item
	registred map[string]bool
	focus     list.ListPos
}

func newNoteListAbstraction() *noteListAbstraction {
	return &noteListAbstraction{
		widgets:   make([]*Item, 0),
		registred: make(map[string]bool, 0),
		focus:     0,
	}
}

func (w *noteListAbstraction) Add(item *Item) int {
	if w.registred[item.ID] {
		// Won't be addressed until we want several clients to run on the same account.
		// The list refreshing is done by restarting the application.
		panic("TODO: update the item proprely")
	}

	w.widgets = append(w.widgets, item)
	w.registred[item.ID] = true
	return len(w.widgets)
}

func (w *noteListAbstraction) Sort(field string) bool {
	switch field {
	case "name":
		sort.Slice(w.widgets, func(i, j int) bool {
			return w.widgets[i].abstraction.Note.Title < w.widgets[j].abstraction.Note.Title
		})
	case "client_updated_at":
		sort.Slice(w.widgets, func(i, j int) bool {
			return w.widgets[i].abstraction.Note.UpdatedAt().After(w.widgets[j].abstraction.Note.UpdatedAt())
		})
	default:
		return false
	}
	return true
}

func (w *noteListAbstraction) ItemAt(i int) *Item {
	return w.widgets[i]
}

func (w *noteListAbstraction) First() list.IWalkerPosition {
	if len(w.widgets) == 0 {
		return nil
	}
	return list.ListPos(0)
}

func (w *noteListAbstraction) Last() list.IWalkerPosition {
	if len(w.widgets) == 0 {
		return nil
	}
	return list.ListPos(len(w.widgets) - 1)
}

func (w *noteListAbstraction) Length() int {
	return len(w.widgets)
}

func (w *noteListAbstraction) At(pos list.IWalkerPosition) gowid.IWidget {
	var res gowid.IWidget
	ipos := int(pos.(list.ListPos))
	if ipos >= 0 && ipos < w.Length() {
		res = w.widgets[ipos]
	}
	return res
}

func (w *noteListAbstraction) Focus() list.IWalkerPosition {
	return w.focus
}

func (w *noteListAbstraction) SetFocus(focus list.IWalkerPosition, app gowid.IApp) {
	w.focus = focus.(list.ListPos)
}

func (w *noteListAbstraction) Next(ipos list.IWalkerPosition) list.IWalkerPosition {
	pos := ipos.(list.ListPos)
	if int(pos) == w.Length()-1 {
		return list.ListPos(-1)
	}
	return pos + 1
}

func (w *noteListAbstraction) Previous(ipos list.IWalkerPosition) list.IWalkerPosition {
	pos := ipos.(list.ListPos)
	if pos-1 == -1 {
		return list.ListPos(-1)
	}
	return pos - 1
}
