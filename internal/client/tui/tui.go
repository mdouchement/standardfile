package tui

import (
	"time"

	"github.com/gcla/gowid"
	"github.com/gcla/gowid/widgets/columns"
	"github.com/gcla/gowid/widgets/framed"
	"github.com/gcla/gowid/widgets/null"
	"github.com/gcla/gowid/widgets/pile"
	"github.com/gcla/gowid/widgets/styled"
	"github.com/gcla/gowid/widgets/text"
	"github.com/gdamore/tcell/v2"
	"github.com/pkg/errors"
)

// A TUI is a text-based interface.
type TUI struct {
	App    *gowid.App
	SortBy string
	list   *NoteList
	editor *framed.Widget
	status *text.Widget
}

// New returns a new TUI.
func New() (*TUI, error) {
	ui := new(TUI)
	ui.SortBy = "name"

	app, err := gowid.NewApp(layout(ui))
	if err != nil {
		return ui, errors.Wrap(err, "could not create application widgets")
	}

	ui.App = app
	return ui, nil
}

// Run starts the application and thus the event loop.
func (ui *TUI) Run() {
	ui.App.MainLoop(gowid.UnhandledInputFunc(ui.unhandled))
}

// Cleanup cleans the application properly (in case of panic).
func (ui *TUI) Cleanup() {
	ui.App.GetScreen().Fini() // Cleanup tcell screen's objects
}

// Register registers an item.
func (ui *TUI) Register(i *Item) {
	ui.list.Register(i)
}

// SortItems sorts the items on ui.SortBy.
func (ui *TUI) SortItems() {
	ui.list.Sort(ui.SortBy)
}

// DisplayStatus displays a message in the status bar (aka notifications).
func (ui *TUI) DisplayStatus(message string) {
	ui.App.Run(gowid.RunFunction(func(app gowid.IApp) { // nolint:errcheck
		ui.status.SetText(message, ui.App)
	}))
	go func() {
		timer := time.NewTimer(1200 * time.Millisecond)
		<-timer.C
		ui.App.Run(gowid.RunFunction(func(app gowid.IApp) { // nolint:errcheck
			ui.status.SetText("", ui.App)
		}))
	}()
}

////////////////////
//                //
// Layout         //
//                //
////////////////////

func layout(ui *TUI) gowid.AppArgs {
	ui.list = NewNoteList(ui)
	ui.editor = framed.NewUnicode(null.New())
	ui.status = text.New("")

	notePane := columns.New([]gowid.IContainerWidget{
		&gowid.ContainerWidget{
			IWidget: styled.New(framed.NewUnicode(ui.list), gowid.MakePaletteRef("mainpane")),
			D:       gowid.RenderWithWeight{W: 1},
		},
		&gowid.ContainerWidget{
			IWidget: styled.New(ui.editor, gowid.MakePaletteRef("mainpane")),
			D:       gowid.RenderWithWeight{W: 8},
		},
	})

	main := pile.New([]gowid.IContainerWidget{
		&gowid.ContainerWidget{IWidget: notePane, D: gowid.RenderWithWeight{W: 20}},
		&gowid.ContainerWidget{
			IWidget: styled.New(framed.NewUnicode(ui.status), gowid.MakePaletteRef("mainpane")),
			D:       gowid.RenderWithWeight{W: 2},
		},
	})

	return gowid.AppArgs{
		View: main,
		Palette: &gowid.Palette{
			"mainpane": gowid.MakePaletteEntry(gowid.ColorLightGray, gowid.ColorBlack),
			// List style
			"normal":  gowid.MakePaletteEntry(gowid.ColorLightGray, gowid.ColorBlack),
			"focused": gowid.MakePaletteEntry(gowid.ColorBlack, gowid.ColorRed),
		},
		Log: NewLogger(),
	}
}

////////////////////
//                //
// Events         //
//                //
////////////////////

func (ui *TUI) unhandled(app gowid.IApp, ev any) bool {
	evk, ok := ev.(*tcell.EventKey)
	if !ok {
		return false
	}

	handled := false

	switch evk.Key() {
	case tcell.KeyCtrlQ:
		handled = true
		app.Quit()
	}

	return handled
}
