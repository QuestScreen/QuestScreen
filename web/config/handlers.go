package config

func (o *item) Edited() {
	o.editIndicator.Set("visible")
}

func (o *view) Edited() {
	// TODO
}
