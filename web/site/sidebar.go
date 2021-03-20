package site

func (o *pageMenuEntry) clicked() {
	if o.active.Get() {
		sidebar.expanded.Set(!sidebar.expanded.Get())
		return
	}
	for cIndex := 0; cIndex < sidebar.items.Len(); cIndex++ {
		c := sidebar.items.Item(cIndex)
		for eIndex := 0; eIndex < c.items.Len(); eIndex++ {
			e := c.items.Item(eIndex)
			e.active.Set(o == e)
		}
	}
	loadView(o.view, o.parent, o.name)
}

type PageControls int

const (
	NoControls PageControls = iota
	CommitControls
	EndControls
)

func setTitle(caption, subtitle string, controls PageControls) {
	if subtitle == "" {
		top.Subtitle.Set(caption)
	} else {
		top.Subtitle.Set(caption + ": " + subtitle)
	}
	top.pageKind.Set(int(controls))
}
