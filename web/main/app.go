package main

import (
	"github.com/QuestScreen/QuestScreen/shared"
	"github.com/QuestScreen/QuestScreen/web"
	"github.com/QuestScreen/QuestScreen/web/datasets"
	"github.com/QuestScreen/QuestScreen/web/info"
	"github.com/QuestScreen/QuestScreen/web/server"
	"github.com/QuestScreen/QuestScreen/web/site"
	api "github.com/QuestScreen/api/web/server"
)

// App is the single-page application managing the web interface.
type App struct {
	shared.Data
	infoPage   *info.Page
	backButton web.BackButtonKind
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
	site.Page.Set(a.infoPage)
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

// ShowDatasets implements the respective controller method for the title nav.
func (a *App) ShowDatasets() {
	site.Page.Set(datasets.NewBase(&a.Data))
}

// SetTitle implements web.PageIF
func (a *App) SetTitle(caption, subtitle string, bb web.BackButtonKind) {
	a.backButton = bb
	site.TitleContent.Title.Set(caption)
	site.TitleContent.Subtitle.Set(subtitle)
	if bb == web.NoBackButton {
		site.TitleContent.BackButtonCaption.Set("")
		site.TitleContent.BackButtonEmpty.Set(true)
	} else {
		if bb == web.BackButtonBack {
			site.TitleContent.BackButtonCaption.Set("Back")
		} else {
			site.TitleContent.BackButtonCaption.Set("Leave")
		}
		site.TitleContent.BackButtonEmpty.Set(false)
	}
}
