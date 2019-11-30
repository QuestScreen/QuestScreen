tmpl.app = {
	groupEntry: new Template("#tmpl-app-group-entry",
			function(app, name, groupIndex) {
		let link = this.querySelector(".pure-menu-link");
		link.textContent = name;
		link.addEventListener('click', app.selectGroup.bind(app, groupIndex));
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

	setPage(content) {
		let page = document.querySelector("#page");
		let newPage = page.cloneNode(false);
		newPage.appendChild(content);
		page.parentNode.replaceChild(newPage, page);
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
		this.setPage(page);
		document.querySelector("#title").textContent = app.groups[index].name;
		for(let [index, entry] of
				document.querySelectorAll(".rp-menu-group-entry").entries()) {
			if (index == this.activeGroup) {
				entry.classList.add("pure-menu-selected");
			} else {
				entry.classList.remove("pure-menu-selected");
			}
		}
	}

	async showConfig() {
		this.setPage(this.cfgPage.ui());
	}

	regenGroupListUI() {
		let groupList = document.querySelector("#menu-groups");
		while (groupList.firstChild && !groupList.firstChild.remove());
		for (const [index, group] of this.groups.entries()) {
			let entry = tmpl.app.groupEntry.render(app, group.name, index);
			groupList.appendChild(entry);
		}
		if (!this.groupDropdownHandler) {
			this.groupDropdownHandler = new DropdownHandler(groupList.parentNode);
		} else {
			this.groupDropdownHandler.hide();
		}
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
		this.regenGroupListUI();
		if (this.activeGroup != -1) {
			this.selectGroup(this.activeGroup);
		}
		this.cfgPage = new ConfigPage(this);
		document.querySelector("#show-config").addEventListener(
			"click", this.showConfig.bind(this));
	}
}

let app = new App();