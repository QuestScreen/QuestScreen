package herolist

// ItemClicked implements the controller of controls.Dropdown
func (w *Widget) ItemClicked(index int) bool {
	if w.Controller != nil {
		return w.Controller.switchHero(index)
	}
	return true
}

func (w *Widget) allClicked() {
	if w.Controller != nil {
		w.allState.Set(w.Controller.switchAll())
	}
}
