package main

type controlCh struct {
	Exit         chan struct{}
	ModuleUpdate chan struct{ index int }
}

func newControlCh() *controlCh {
	return &controlCh{
		Exit:         make(chan struct{}),
		ModuleUpdate: make(chan struct{ index int }),
	}
}
