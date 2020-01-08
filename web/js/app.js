tmpl.app = {
	groupEntry: new Template("#tmpl-app-group-entry",
			function(app, name, groupIndex) {
		const link = this.querySelector(".pure-menu-link");
		link.textContent = name;
		link.addEventListener('click', e => {
			const groupMenuBtn = document.querySelector("#open-group-menu");
			groupMenuBtn.click();
			groupMenuBtn.blur();
			app.setGroup(groupIndex);
			e.preventDefault();
		});
		// return directly the <li> element.
		return this.children[0];
	}),
	pageMenuEntry: new Template("#tmpl-app-page-menu-entry",
			function(app, handler, name, icon, parent) {
		const link = this.querySelector(".pure-menu-link");
		link.insertBefore(document.createTextNode(" " + name), link.firstChild);
		if (parent != null) {
			const parentSpan = document.createElement("span");
			parentSpan.classList.add("submenu-parent-name");
			parentSpan.textContent = " " + parent + " –";
			link.insertBefore(parentSpan, link.firstChild);
			link.classList.add("submenu-link");
		}
		const i = document.createElement("i");
		i.classList.add("fas");
		i.classList.add(icon);
		link.insertBefore(i, link.firstChild);

		link.addEventListener("click", e => {
			if (app.handlePageMenuClick(link.parentNode)) {
				handler();
			}
			e.preventDefault();
		});
		return this.children[0];
	}),
	dataMenu: new Template("#tmpl-app-data-menu",
			function(app, controller) {
		controller.activeMenuEntry = null;
		const list = this.querySelector(".pure-menu-list");
		const baseEntry = tmpl.app.pageMenuEntry.render(
			app, controller.viewBase.bind(controller), "Base", "fa-tools");
		list.insertBefore(baseEntry, list.firstChild);
		if (app.activeGroup == -1) {
			controller.activeMenuEntry = baseEntry;
		}

		let curLast = list.querySelector(".config-menu-system-heading");
		for (const [index, system] of app.systems.entries()) {
			const entry = tmpl.app.pageMenuEntry.render(app,
				controller.viewSystem.bind(controller, system), system.name, "fa-book");
			// Safari doesn't support firstElementChild on DocumentFragment
			curLast = curLast.parentNode.insertBefore(entry, curLast.nextSibling);
			if (app.activeGroup == -1 && index == 0) {
				controller.activeMenuEntry = entry;
			}
		}

		curLast = this.querySelector(".config-menu-group-heading");
		for (const [index, group] of app.groups.entries()) {
			const entry = tmpl.app.pageMenuEntry.render(app,
				controller.viewGroup.bind(controller, group), group.name, "fa-users");
			curLast = curLast.parentNode.insertBefore(entry, curLast.nextSibling);
			if (index == app.activeGroup) {
				controller.activeMenuEntry = entry;
			}
			for (const scene of group.scenes) {
				const sceneEntry = tmpl.app.pageMenuEntry.render(app,
					controller.viewScene.bind(controller, group, scene), scene.name, "fa-image",
					group.name);
				curLast = curLast.parentNode.insertBefore(
					sceneEntry, curLast.nextSibling);
			}
		}
	}),
};

const datakind = {
	Base: "Base", System: "System", Group: "Group", Scene: "Scene"
}

class DataPage {
	constructor(app, rootPath, viewgen, name, fetchOnLoad) {
		this.app = app;
		this.rootPath = rootPath;
		this.viewgen = viewgen;
		this.name = name;
		this.fetchOnLoad = fetchOnLoad;
	}

	async loadView(subpath, kind, subtitle) {
		const url = this.rootPath + subpath;

		if (this.fetchOnLoad) {
			const cfgData = await App.fetch(url, "GET", null);
			const view = this.viewgen(this.app, kind, cfgData, url);
			this.app.setPage(view.ui(cfgData));
		} else {
			const view = this.viewgen(this.app, kind, url);
			this.app.setPage(view.ui());
		}
		this.app.setTitle(kind + " " + this.name, subtitle);
	}

	async viewBase() {
		this.loadView("/base", datakind.Base, "");
	}

	async viewSystem(system) {
		this.loadView("/systems/" + system.id, datakind.System, system.name);
	}

	async viewGroup(group) {
		this.loadView("/groups/" + group.id, datakind.Group, group.name);
	}

	async viewScene(group, scene) {
		this.loadView("/groups/" + group.id + "/" + scene.id,
									datakind.Scene, group.name + " " + scene.name);
	}

	genMenu() {
		return tmpl.app.dataMenu.render(this.app, this);
	}
}

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
		const body = content == null ? null : JSON.stringify(content);
		const headers = { 'X-Clacks-Overhead': 'GNU Terry Pratchett'};
		if (body != null) {
			headers['Content-Type'] = 'application/json';
		}
		const response = await fetch(url, {
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

	setMenu(content) {
		const pagemenu = document.querySelector("#pagemenu");
		const newMenu = pagemenu.cloneNode(false);
		newMenu.appendChild(content);
		pagemenu.parentNode.replaceChild(newMenu, pagemenu);
	}

	setPage(content) {
		const page = document.querySelector("main");
		const newPage = page.cloneNode(false);
		newPage.appendChild(content);
		page.parentNode.replaceChild(newPage, page);
	}

	setTitle(caption, subtitle) {
		document.querySelector("#title").textContent = caption;
		document.querySelector("#subtitle").textContent = subtitle;
	}

	handlePageMenuClick(item) {
		const pagemenu = document.querySelector("#pagemenu");
		if (item.classList.contains("pure-menu-active")) {
			if (pagemenu.classList.contains("pagemenu-expanded")) {
				pagemenu.classList.remove("pagemenu-expanded");
			} else {
				pagemenu.classList.add("pagemenu-expanded");
			}
			return false;
		} else {
			for (const other of pagemenu.querySelectorAll(".pure-menu-item")) {
				other.classList.remove("pure-menu-active");
			}
			item.classList.add("pure-menu-active");
			pagemenu.classList.remove("pagemenu-expanded");
			return true;
		}
	}

	async setGroup(groupIndex) {
		const response = await App.fetch(
			"/state", "POST", {action: "setgroup", index: groupIndex});
		this.updateViewFromStateResponse(response);
	}

	updateViewFromStateResponse(response) {
		if (!Array.isArray(response.modules) ||
				response.modules.length != this.modules.length) {
			throw Error(
					"Invalid response structure (resp.modules not an array or wrong length)");
		} else if (response.activeGroup < 0 ||
				response.activeGroup >= this.groups.length ||response.activeScene < 0 ||
				response.activeScene >= this.groups[response.activeGroup].scenes.length) {
			throw Error("Invalid response (resp.activeScene outside of group scene range)")
		}
		this.activeGroup = response.activeGroup;
		this.setTitle(this.groups[response.activeGroup].name, "");
		this.setMenu(this.statePage.genMenu(response.activeScene));
		this.statePage.setSceneData(response.modules);
		for(const [index, entry] of
				document.querySelectorAll(".rp-menu-group-entry").entries()) {
			if (index == this.activeGroup) {
				entry.classList.add("pure-menu-selected");
			} else {
				entry.classList.remove("pure-menu-selected");
			}
		}
	}

	async showConfig() {
		this.setMenu(this.cfgPage.genMenu());
		if (this.cfgPage.activeMenuEntry == null) {
			const empty = document.createElement("p");
			empty.textContent = "Select group or system";
			const article = document.createElement("article");
			article.appendChild(empty);
			this.setPage(article);
		} else {
			this.cfgPage.activeMenuEntry.querySelector("a").click();
		}
	}

	async showDatasets() {
		this.setMenu(this.datasetPage.genMenu());
		this.datasetPage.viewBase();
	}

	async toggleHeader(link) {
		const header = document.querySelector("header");
		const classes = link.children[0].classList;
		if (this.headerHeight) {
			classes.remove("fa-angle-down");
			classes.add("fa-angle-up");
			header.addEventListener("transitionend", e => {
				header.style.height = "";
				header.style.paddingBottom = "";
				header.style.overflow = "";
			}, {once: true});
			header.style.height = this.headerHeight + "px";
			this.headerHeight = false;
		} else {
			classes.remove("fa-angle-up");
			classes.add("fa-angle-down");
			this.headerHeight = header.offsetHeight;
			// no transition since height was 'auto' before
			header.style.height = this.headerHeight + "px";
			header.style.paddingBottom = 0;
			header.style.overflow = "hidden";
			header.offsetWidth; // forces repaint
			header.style.height = 0;
		}
	}

	regenGroupListUI() {
		const groupList = document.querySelector("#menu-groups");
		while (groupList.firstChild && !groupList.firstChild.remove());
		for (const [index, group] of this.groups.entries()) {
			const entry = tmpl.app.groupEntry.render(app, group.name, index);
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
		const returned = await App.fetch("/static", "GET", null);
		for (const module of returned.modules) {
			if (this.controllers.hasOwnProperty(module.id)) {
				module.controller = this.controllers[module.id];
				this.modules.push(module);
			} else {
				console.error("Missing controller for module \"%s\"", module.id);
			}
		}
		this.fonts = returned.fonts;
		this.plugins = returned.plugins;

		const config = await App.fetch("/datasets", "GET", null);
		this.systems = config.systems;
		this.groups = config.groups;
		this.activeGroup = -1;
		this.regenGroupListUI();

		this.statePage = new StatePage(this);
		const stateResp = await App.fetch("/state", "GET", null);
		if (stateResp.activeGroup != -1) {
			this.updateViewFromStateResponse(stateResp);
		}

		this.cfgPage = new DataPage(this, "/config",
				(app, _, data, url) => new ConfigView(app, data, url), "Configuration",
				true);
		this.datasetPage = new DataPage(this, "/datasets", genDatasetView,
				"Dataset", false);
		document.querySelector("#show-config").addEventListener(
				"click", e => {
					e.target.blur();
					this.showConfig();
					e.preventDefault();
				});
		document.querySelector("#header-toggle").addEventListener(
				"click", e => {
					this.toggleHeader(e.currentTarget);
					e.preventDefault();
				});
		document.querySelector("#show-datasets").addEventListener(
			"click", e => {
				this.showDatasets();
				e.preventDefault();
			});
	}
}

const app = new App();