package main

type controlCh struct {
	Exit chan struct{}
	Draw chan struct{}
}

func newControlCh() *controlCh {
	return &controlCh{
		Exit: make(chan struct{}),
		Draw: make(chan struct{}),
	}
}
