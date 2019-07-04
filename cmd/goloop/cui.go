package main

import (
	"github.com/icon-project/goloop/common/log"
	"github.com/jroimartin/gocui"
)

var (
	CuiQuitKeyEvtFunc  = func(g *gocui.Gui, v *gocui.View) error { return gocui.ErrQuit }
	CuiQuitUserEvtFunc = func(g *gocui.Gui) error { return gocui.ErrQuit }
	CuiNilUserEvtFunc  = func(g *gocui.Gui) error { return nil }
)

func NewCui() (*gocui.Gui, <-chan bool) {
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicln(err)
	}

	g.SetManagerFunc(func(g *gocui.Gui) error {
		maxX, maxY := g.Size()
		if v, err := g.SetView("main", -1, -1, maxX, maxY); err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			v.Wrap = true
			v.Overwrite = true
		}
		return nil
	})

	if err = g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, CuiQuitKeyEvtFunc); err != nil {
		g.Close()
		log.Panicln(err)
	}
	termCh := make(chan bool)
	go func() {
		defer close(termCh)
		if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
			log.Panicln(err)
		}
		log.Println("gui MainLoop terminate")
	}()
	return g, termCh
}

func TermGui(g *gocui.Gui, termCh <-chan bool) {
	g.Update(CuiQuitUserEvtFunc)
	log.Println("waiting gui terminate")
	select {
	case <-termCh:
		log.Println("gui terminated")
	}
	g.Close()
}
