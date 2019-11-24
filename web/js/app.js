tmpl.app = {
	menuGroupEntry: new Template("#tmpl-app-menu-group-entry",
			function(handleSelect, handleSettings, groupName) {
		let link = this.querySelector("a.pure-menu-link");
		link.addEventListener('click', handleSelect);
		link.href = "#";
		link.textContent = groupName;
		let cog = this.querySelector("a.settings-link");
		cog.addEventListener('click', handleSettings);
		return this;
	}),

	menu: new Template("#tmpl-app-menu", function(app) {
		let curLast = this.querySelector("#rp-menu-group-heading");
		for (let [index, group] of app.groups.entries()) {
			let entry = tmpl.app.menuGroupEntry.render(
					app.selectGroup.bind(app, index),
					app.groupSettings.bind(app, group),
					group.name
			);
			// Safari doesn't support firstElementChild on DocumentFragment
			curLast = curLast.parentNode.insertBefore(entry.children[0], curLast.nextSibling);
		}
		return this;
	}),

	state: {
		module: new Template("#tmpl-app-state-module",
				function(app, moduleIndex, state) {
			let wrapper = this.querySelector(".state-module-content");
			this.querySelector(".state-module-name").textContent =
					app.modules[moduleIndex].name;
			wrapper.appendChild(
					app.modules[moduleIndex].controller.ui(app, state));
			return this;
		}),

		page: new Template("#tmpl-app-state-page",
					function(app, moduleStates) {
			this.querySelector("#group-heading").textContent =
					app.groups[app.activeGroup].name;
			let stateWrapper = this.querySelector("#module-state-wrapper");
			for (let [index, state] of moduleStates.entries()) {
				stateWrapper.appendChild(
						tmpl.app.state.module.render(app, index, state));
			}
			return this;
		})
	}
};

class App {
	constructor() {
		this.controllers = {};
		this.modules = [];
		this.systems = [];
		this.groups = [];
		this.fonts = [];
	}

	/* registers a controller. must be called before init(). */
	registerController(controller) {
		this.controllers[controller.id] = controller;
	}

	static async fetch(url, method, content) {
		let body = content == null ? null : JSON.stringify(content);
		let headers = { 'X-Clacks-Overhead': 'GNU Terry Pratchett'};
		if (body != null) {
			headers['Content-Type'] = 'application/json';
		}
		let response = await fetch(url, {
				method: method, mode: 'no-cors', cache: 'no-cache',
				credentials: 'omit', redirect: 'follow', referrer: 'no-referrer',
				headers: headers, body: body,
		});
		if (response.ok) {
			if (response.status == 200) {
				return await response.json();
			} else return null;
		} else {
			throw new Error("failed to fetch " + url);
		}
	}

	setMain(content) {
		let main = document.querySelector("#main");
		let newMain = main.cloneNode(false);
		newMain.appendChild(content);
		main.parentNode.replaceChild(newMain, main);
		initDropdowns();
	}

	async selectGroup(index) {
		let moduleStates = await App.fetch("/groups/" + this.groups[index].id);
		if (!Array.isArray(moduleStates) ||
			moduleStates.length != this.modules.length) {
			throw Error(
					"Invalid response structure (not an array or wrong length");
		}
		this.activeGroup = index;
		let page = tmpl.app.state.page.render(this, moduleStates);
		this.setMain(page);
		for(let [index, entry] of
				document.querySelectorAll(".rp-menu-group-entry").entries()) {
			if (index == this.activeGroup) {
				entry.classList.add("pure-menu-selected");
			} else {
				entry.classList.remove("pure-menu-selected");
			}
		}
	}

	async groupSettings(group) {
		let url = "/config/groups/" + group.id;
		let cfgData = await App.fetch(url, "GET", null);
		let cfgPage = new ConfigPage(this, cfgData, url);
		let page = cfgPage.ui(group.name + " Settings", cfgData);
		this.setMain(page);
	}

	/* queries the global config from the server and initializes the app. */
	async init() {
		let returned = await App.fetch("/app", "GET", null);
		for (const module of returned.modules) {
			if (this.controllers.hasOwnProperty(module.id)) {
				module.controller = this.controllers[module.id];
				this.modules.push(module);
			} else {
				console.error("Missing controller for module \"%s\"", module.id);
			}
		}
		this.systems = returned.systems;
		this.groups = returned.groups;
		this.fonts = returned.fonts;
		this.activeGroup = returned.activeGroup;

		let menu = document.querySelector("#menu");
		let renderedMenu = menu.cloneNode(false);
		renderedMenu.appendChild(tmpl.app.menu.render(this));
		menu.parentNode.replaceChild(renderedMenu, menu);

		if (this.activeGroup != -1) {
			this.selectGroup(this.activeGroup);
		}
	}
}

let app = new App();