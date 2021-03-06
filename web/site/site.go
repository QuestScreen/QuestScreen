// Package site implements the site all of the UI runs in.
// It contains the HTML skeleton and provides access to top-level UI elements
// like the title bar and the sidebar.
//
// The UI's terminology is as follows:
// - The "site" is the HTML document loading all the JavaScript and providing
//   the singleton top level UI elements like the header, the title bar and the
//   side bar. It provides an interface to these elements for pages.
//   The site also contains any global data.
// - A "page" is a collection of views for a certain purpose. The main active
//   role of a page is to define the content of the sidebar, which provides
//   access to all views of the page. Each page is accessible view its dedicated
//   button in the site header.
// - A "view" is a collection of UI elements shown in the main part of the site.
//   It belongs to a page and is typically accessed via the page's side bar.
//   A view defines the current title.
package site

import (
	"github.com/QuestScreen/QuestScreen/shared"
	"github.com/QuestScreen/QuestScreen/web"
	"github.com/QuestScreen/QuestScreen/web/comms"
	"github.com/QuestScreen/api/server"
	askew "github.com/flyx/askew/runtime"
)

// View describes a collection of UI elements that fill the main part of the
// site. It belongs to a page and is typically accessed via the site bar.
//
// The view itself should be stateless. Only when the UI elements are actually
// generated via GenerateUI may a private object holding some state be
// instantiated. This is because all views of a page are queried at once to be
// able to generate the sidebar.
type View interface {
	// Title returns the label that should be written to the title bar when this
	// view is being displayed.
	Title() string
	// ID returns the identifier of the view. Defines view equality.
	// Necessary because we want to identify and select the previous view when
	// regenerating views.
	ID() string
	// IsChild indicates whether this view should be shown as a child of the most
	// recent view with IsChild() == false in the sidebar.
	IsChild() bool
	// SwitchTo instructs the view that the user wants to switch to it.
	// It will make all necessary preparations and return the UI widget that
	// represents the view.
	SwitchTo(ctx server.Context) askew.Component
}

// ViewCollection describes a set of views.
// It can optionally have a title.
type ViewCollection struct {
	// Title may be empty if this group has no title.
	Title string
	// Items contains all views part of this collection.
	Items []View
}

// Page describes a collection of views and the sidebar used to navigate between
// them.
type Page interface {
	// Title returns the label that should be written to the title bar when a
	// view of this page is being displayed. It is used as prefix for the view's
	// title.
	Title() string
	// GenViews generates the list of views of the page and returns it.
	// The views are organized in collections. This affects their rendering in the
	// sidebar; each collection will be rendered with its title above it.
	//
	// If the returned list contains only a single view, the sidebar is left
	// empty and only the title of the page is used in the title bar.
	GenViews() []ViewCollection
	// IconOffset defines the offset of the icon index used for the views of this
	// page. The icon index for a view collection is selected based on its index.
	IconOffset() int
}

// PageEditHandler is a callback that is used by commitable pages to indicate
// whether their state differs from the currently commited state.
type PageEditHandler interface {
	SetEdited(value bool)
}

// CommitablePage is a page whose views have a UI state which can be commited or
// reset. Moving away from a CommitablePage shall prompt the user that data will
// be discarded if it has not been saved.
//
// A CommitablePage will have „Save“ and „Reset“ button in its title bar which
// will call the page's Commit() / Reset() functions.
type CommitablePage interface {
	Page
	// RegisterEditHandler registers a handler with the page which will be called
	// whenever the state page switches between commited and uncommited.
	RegisterEditHandler(handler PageEditHandler)
	// Commit instructs the page to commit the current state.
	// The page is assumed to be in commited state afterwards.
	Commit()
	// Reset instructs the page to reset the current state to the last commited /
	// initial state.
	Reset()
}

// EndablePage is a page that can be „ended“.
type EndablePage interface {
	Page
	// End ends the entity the current page visualizes.
	End()
}

// PageKind describes the known kinds of pages.
type PageKind int

const (
	// InfoPage ist the info page shown at startup or when no session is active.
	InfoPage PageKind = iota
	// SessionPage is the page shown during a session.
	SessionPage
	// ConfigPage is the page for customizing configuration.
	ConfigPage
	// DataPage is the page for manipulating systems & groups.
	DataPage
)

type siteContent struct {
	shared.State
	pages   [4]Page
	curPage PageKind
}

func (sc *siteContent) page() Page {
	return sc.pages[int(sc.curPage)]
}

var site siteContent

func (sc *siteContent) showHome() {
	if sc.ActiveGroup == -1 {
		sc.curPage = InfoPage
		Refresh("")
	} else {
		sc.curPage = SessionPage
		Refresh(web.Data.Groups[sc.ActiveGroup].Scenes[sc.ActiveScene].ID)
	}
}

func (sc *siteContent) showConfig() {
	sc.curPage = ConfigPage
	Refresh("")
}

func (sc *siteContent) showDatasets() {
	sc.curPage = DataPage
	Refresh("")
}

func (sc *siteContent) commitablePageReset() {
	sc.page().(CommitablePage).Reset()
}

func (sc *siteContent) commitablePageCommit() {
	go sc.page().(CommitablePage).Commit()
}

func (sc *siteContent) endablePageEnd() {
	go sc.page().(EndablePage).End()
}

func (sc *siteContent) SetEdited(value bool) {
	top.commitablePageEdited.Set(value)
}

// ShowHome shows the info page if no session is in progress.
// It shows the current session state if one is in progress.
func ShowHome() {
	site.showHome()
}

// RegisterPage registers the page implementation for the given page kind with
// the site. This must be done before calling Boot().
func RegisterPage(kind PageKind, page Page) {
	site.pages[int(kind)] = page
	if commitable, ok := page.(CommitablePage); ok {
		commitable.RegisterEditHandler(&site)
	}
}

// Boot starts the site and loads the info page.
func Boot(headerDisabled bool) {
	top.Disabled.Set(headerDisabled)
	top.Controller = &site
	top.homeLabel.Set("Home")
	site.ActiveGroup = -1
}

// Refresh must be called from a view when it modified system, group or
// scene data so that the side bar needs to be updated.
//
// Refresh will load the view with the given id after regenerating the
// sidebar, or the first view if no view with that id is found.
func Refresh(id string) {
	// TODO: ask commitable page whether it is edited and prompt the user.
	sidebar.items.DestroyAll()
	viewColls := site.page().GenViews()
	top.Title.Set(site.page().Title())
	if site.curPage == InfoPage {
		// single-view page. leave the sidebar empty, display the view.
		v := viewColls[0].Items[0]
		sidebar.Disabled.Set(true)
		loadView(v, "", "")
		return
	}

	iconOffset := site.page().IconOffset()
	var newSelectedEntry *pageMenuEntry
	for cIndex, c := range viewColls {
		coll := newSidebarColl(c.Title)
		var parentName string
		for _, v := range c.Items {
			var entry *pageMenuEntry
			if v.IsChild() {
				entry = newPageMenuEntry(v.Title(), parentName, v, iconOffset+cIndex+1)
			} else {
				entry = newPageMenuEntry(v.Title(), "", v, iconOffset+cIndex)
				parentName = v.Title()
			}
			coll.items.Append(entry)
			if newSelectedEntry == nil || v.ID() == id {
				newSelectedEntry = entry
			}
		}
		sidebar.items.Append(coll)
	}
	sidebar.Disabled.Set(false)
	newSelectedEntry.active.Set(true)
	loadView(newSelectedEntry.view, newSelectedEntry.parent,
		newSelectedEntry.view.Title())
}

func loadView(v View, parent, name string) {
	controls := NoControls
	switch site.curPage {
	case ConfigPage:
		controls = CommitControls
		top.commitablePageEdited.Set(false)
	case SessionPage:
		controls = EndControls
	}

	sidebar.expanded.Set(false)
	go func(controls PageControls) {
		content.Set(v.SwitchTo(&comms.ServerState{&site.State, ""}))
		if parent == "" {
			setTitle(name, "", controls)
		} else {
			setTitle(parent, name, controls)
		}
	}(controls)
}

func UpdateSession(groupIndex, sceneIndex int) {
	if (site.State.ActiveGroup == -1) != (groupIndex == -1) {
		if groupIndex == -1 {
			top.homeLabel.Set("Home")
		} else {
			top.homeLabel.Set("Session")
		}
	}
	site.ActiveGroup, site.ActiveScene = groupIndex, sceneIndex
}

func State() *shared.State {
	return &site.State
}
