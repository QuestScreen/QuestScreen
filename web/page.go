package web

// BackButtonKind describes the kind of action the back button in the title bar
// is used for.
type BackButtonKind int

const (
	// NoBackButton defines that the back button is invisible
	NoBackButton BackButtonKind = iota
	// BackButtonBack defines that the back button returns the user to the
	// previous page
	BackButtonBack
	// BackButtonLeave defines that the back button leaves the current session.
	BackButtonLeave
)

// PageIF is the interface to the page managed by the main app.
type PageIF interface {
	SetTitle(caption, subtitle string, bb BackButtonKind)
}

// Page contains the implementation of the PageIF (i.e. main.App)
var Page PageIF
