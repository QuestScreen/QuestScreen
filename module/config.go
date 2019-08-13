package module

// SelectableFont is used to allow the user to select a font family.
type SelectableFont struct {
	Family string
	Index  int32 `yaml:"-"`
}
