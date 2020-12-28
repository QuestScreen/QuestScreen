package main

import (
	"fmt"

	"github.com/QuestScreen/QuestScreen/shared"
	"github.com/QuestScreen/QuestScreen/web"
	"github.com/QuestScreen/QuestScreen/web/datasets"
	"github.com/QuestScreen/QuestScreen/web/info"
	"github.com/QuestScreen/QuestScreen/web/server"
	api "github.com/QuestScreen/api/web/server"
	"github.com/gopherjs/gopherjs/js"
)

// App is the single-page application managing the web interface.
type App struct {
	shared.Data
	infoPage     *info.Page
	headerHeight int
	backButton   web.BackButtonKind
}

// Init initializes the app by querying group and system data from the server.
func (a *App) Init() {
	if err := server.Fetch(api.Get, "/data", nil, &a.Data); err != nil {
		panic(err)
	}
	a.infoPage = info.ConstructInfoPage()
}

// ShowInfo shows the info page.
func (a *App) ShowInfo() {
	Page.Set(a.infoPage)
}

// BackButtonClicked implements the respective controller method for the title bar.
func (a *App) BackButtonClicked() {
	if a.backButton == web.BackButtonLeave {
		// TODO
	} else {
		// TODO: back to group if one is loaded
		a.ShowInfo()
	}
}

// HeaderToggleClicked implemnts the respective controller method for the title bar.
func (a *App) HeaderToggleClicked(target *js.Object) {
	if a.headerHeight > 0 {
		TitleContent.ToggleOrientation.Set(1)
		Header.Self.Get().Call("addEventListener", func() {
			Header.height.Set("")
			Header.paddingBottom.Set("")
			Header.overflow.Set("")
		}, struct{ once bool }{true})
		Header.height.Set(fmt.Sprintf("%vpx", a.headerHeight))
		a.headerHeight = 0
	} else {
		TitleContent.ToggleOrientation.Set(2)
		a.headerHeight = Header.offsetHeight.Get()
		// no transition since height was 'auto' before
		Header.height.Set(fmt.Sprintf("%vpx", a.headerHeight))
		Header.paddingBottom.Set("0")
		Header.overflow.Set("hidden")
		Header.offsetWidth.Get() // forces repaint
		Header.height.Set("0")
	}
}

func (a *App) showDatasets() {
	Page.Set(datasets.NewBase(&a.Data))
}

// SetTitle implements web.PageIF
func (a *App) SetTitle(caption, subtitle string, bb web.BackButtonKind) {
	a.backButton = bb
	TitleContent.Title.Set(caption)
	TitleContent.Subtitle.Set(subtitle)
	if bb == web.NoBackButton {
		TitleContent.BackButtonCaption.Set("")
		TitleContent.BackButtonEmpty.Set(true)
	} else {
		if bb == web.BackButtonBack {
			TitleContent.BackButtonCaption.Set("Back")
		} else {
			TitleContent.BackButtonCaption.Set("Leave")
		}
		TitleContent.BackButtonEmpty.Set(false)
	}
}
