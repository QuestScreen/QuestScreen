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

func setTitle(caption, subtitle string) {
	if subtitle == "" {
		top.Subtitle.Set(caption)
	} else {
		top.Subtitle.Set(caption + ": " + subtitle)
	}
}
