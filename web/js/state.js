const SelectorKind = Object.freeze({
	atMostOne: Symbol("atMostOne"),
	exactlyOne: Symbol("exactlyOne"),
	multiple: Symbol("multiple")
});

tmpl.state = {
	module: new Template("#tmpl-state-module",
			function(app, moduleIndex, state) {
		const wrapper = this.querySelector(".state-module-content");
		this.querySelector(".state-module-name").textContent =
				app.modules[moduleIndex].name;
		wrapper.appendChild(app.modules[moduleIndex].controller.ui(app, state));
	}),
	scene: new Template("#tmpl-state-scene", function(app, moduleStates) {
		const stateWrapper = this.querySelector("#module-state-wrapper");
		for (const [index, state] of moduleStates.entries()) {
			if (state != null) {
				stateWrapper.appendChild(
						tmpl.state.module.render(app, index, state));
			}
		}
	}),
	menu: new Template("#tmpl-state-menu", function(app, statePage, activeScene) {
		const list = this.querySelector(".pure-menu-list");
		for (const [index, scene] of app.groups[app.activeGroup].scenes.entries()) {
			const entry = tmpl.app.pageMenuEntry.render(app, null,
				statePage.setScene.bind(statePage, index), scene.name, "fa-image");
			if (index == activeScene) {
				entry.classList.add("pure-menu-active");
			}
			list.appendChild(entry);
		}
	})
}

class StatePage {
	constructor(app) {
		this.app = app;
	}

	setSceneData(modules) {
		const page = tmpl.state.scene.render(this.app, modules);
		this.app.setPage(page);

	}

	async setScene(sceneIndex) {
		const sceneResp = await App.fetch("/state", "POST",
				{action: "setscene", index: sceneIndex});
		this.setSceneData(sceneResp.modules);
	}

	genMenu(activeScene) {
		return tmpl.state.menu.render(app, this, activeScene);
	}
}