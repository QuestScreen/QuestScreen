package site

func (o *pageMenuEntry) init(name string, parent string, view View) {
	o.view = view
	o.name, o.parent = name, parent
}

func (o *pageMenuEntry) clicked() {
	if o.active.Get() {
		sidebar.expanded.Set(!sidebar.expanded.Get())
		return
	}
	for _, c := range sidebar.items.items {
		for _, i := range c.items.items {
			i.active.Set(o == i)
		}
	}
	loadView(o.view, o.parent, o.name)
}

func setTitle(caption, subtitle string) {
	top.Title.Set(caption)
	top.Subtitle.Set(subtitle)
	bb := site.page().BackButton()
	if bb == NoBackButton {
		top.BackButtonCaption.Set("")
		top.BackButtonEmpty.Set(true)
	} else {
		if bb == BackButtonBack {
			top.BackButtonCaption.Set("Back")
		} else {
			top.BackButtonCaption.Set("Leave")
		}
		top.BackButtonEmpty.Set(false)
	}
}
