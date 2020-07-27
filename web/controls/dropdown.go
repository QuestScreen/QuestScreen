package controls

// SelectorKind defines how items in a dropdown menu are selected.
type SelectorKind int

const (
	// SelectSingle is used when selecting a new item unselects the previous one.
	SelectSingle SelectorKind = iota
	// SelectMultiple is used when multiple items can be selected at the same time.
	SelectMultiple
)

func (d *Dropdown) init(kind SelectorKind) {
	d.kind = kind
}

func (d *Dropdown) click(index int) {
	if d.Controller != nil {
		newVal := d.Controller.ItemClicked(index)

		if d.kind == SelectMultiple {
			item := d.Items.Item(index)
			item.Selected.Set(newVal)
		} else {
			for i := 0; i < d.Items.Len(); i++ {
				if i == index {
					item := d.Items.Item(i)
					item.Selected.Set(true)
					d.caption.Set(item.caption.Get())
				} else {
					d.Items.Item(i).Selected.Set(false)
				}
			}
		}
	}
}
