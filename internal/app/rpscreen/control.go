package rpscreen

type controlCh struct {
	Exit chan struct{}
	Draw chan struct{}
}

func NewControlCh() *controlCh {
	return &controlCh{
		Exit: make(chan struct{}),
		Draw: make(chan struct{}),
	}
}
