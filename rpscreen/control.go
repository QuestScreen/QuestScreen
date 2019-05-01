package main

type wmEvent int

const (
	wmExit wmEvent = iota
	wmRedraw
)

type controlCh struct {
	WMEvents     chan wmEvent
	ModuleUpdate chan struct{ index int }
}

func newControlCh() *controlCh {
	return &controlCh{
		WMEvents:     make(chan wmEvent),
		ModuleUpdate: make(chan struct{ index int }),
	}
}
