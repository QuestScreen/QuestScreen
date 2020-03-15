const MenuSelect = {
	Base: 0, Active: 1, Previous: 2
}

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
			function(app, controller, handler, name, icon, parent) {
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
			if (app.handlePageMenuClick(controller, link.parentNode)) {
				handler();
			}
			e.preventDefault();
		});
		return this.children[0];
	}),
	dataMenu: new Template("#tmpl-app-data-menu",
			function(app, controller, select) {
		let selectedId = "0";
		if (select == MenuSelect.Previous) {
			if (controller.activeMenuEntry == null) {
				select = MenuSelect.Base;
			} else {
				selectedId = controller.activeMenuEntry.dataset.id;
			}
		}

		let runningId = 0;

		const list = this.querySelector(".pure-menu-list");
		const baseEntry = tmpl.app.pageMenuEntry.render(app, controller,
			controller.viewBase.bind(controller), "Base", "fa-tools");
		baseEntry.dataset.id = runningId++;
		list.insertBefore(baseEntry, list.firstChild);

		let curLast = list.querySelector(".config-menu-system-heading");
		for (const system of app.systems) {
			const entry = tmpl.app.pageMenuEntry.render(app, controller,
				controller.viewSystem.bind(controller, system), system.name, "fa-book");
			entry.dataset.id = runningId++;
			// Safari doesn't support firstElementChild on DocumentFragment
			curLast = curLast.parentNode.insertBefore(entry, curLast.nextSibling);
		}

		curLast = this.querySelector(".config-menu-group-heading");
		for (const [index, group] of app.groups.entries()) {
			const entry = tmpl.app.pageMenuEntry.render(app, controller,
				controller.viewGroup.bind(controller, group), group.name, "fa-users");
			curLast = curLast.parentNode.insertBefore(entry, curLast.nextSibling);
			entry.dataset.id = runningId;
			if (select == MenuSelect.Active && index == app.activeGroup) {
				selectedId = entry.dataset.id;
			}
			runningId++;
			for (const scene of group.scenes) {
				const sceneEntry = tmpl.app.pageMenuEntry.render(app, controller,
					controller.viewScene.bind(controller, group, scene), scene.name, "fa-image",
					group.name);
				sceneEntry.dataset.id = runningId++;
				curLast = curLast.parentNode.insertBefore(
					sceneEntry, curLast.nextSibling);
			}
		}
		controller.activeMenuEntry = list.querySelector("li[data-id='" + selectedId + "']");
		if (controller.activeMenuEntry == null)
			controller.activeMenuEntry = baseEntry;
		controller.activeMenuEntry.classList.add("pure-menu-active");
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

	async loadView(subpath, kind, item) {
		const url = this.rootPath + subpath;

		if (this.fetchOnLoad) {
			const cfgData = await App.fetch(url, "GET", null);
			const view = this.viewgen(this.app, kind, url, cfgData);
			this.app.setPage(view.ui(cfgData));
		} else {
			const view = this.viewgen(this.app, kind, url, item);
			this.app.setPage(view.ui());
		}
		this.titleMain = kind + " " + this.name;
		this.updateTitle(item);
	}

	async viewBase() {
		this.loadView("/base", datakind.Base, null);
	}

	async viewSystem(system) {
		this.loadView("/systems/" + system.id, datakind.System, system);
	}

	async viewGroup(group) {
		this.loadView("/groups/" + group.id, datakind.Group, group);
	}

	async viewScene(group, scene) {
		this.loadView("/groups/" + group.id + "/scenes/" + scene.id, datakind.Scene,
				{group: group, scene: scene,
				 name: group.name + " " + scene.name});
	}

	genMenu(select) {
		return tmpl.app.dataMenu.render(this.app, this, select);
	}

	updateTitle(item) {
		const subtitle = item == null ? "" : item.name;
		this.app.setTitle(this.titleMain, subtitle, false);
	}
}

class App {
	constructor() {
		this.stateControllers = {};
		this.configItemControllers = {};
		this.modules = [];
		this.plugins = [];
		this.systems = [];
		this.groups = [];
		this.fonts = [];
		this.backButtonLeavesGroup = false;
	}

	/* registers a controller for a module state. must be called before init(). */
	registerStateController(controller) {
		this.stateControllers[controller.id] = controller;
	}

	registerConfigItemController(controller) {
		this.configItemControllers[controller.name] = controller;
	}

	static async fetch(url, method, content) {
		const body = content == null ? null : JSON.stringify(content);
		const headers = { 'X-Clacks-Overhead': 'GNU Terry Pratchett'};
		if (body != null) {
			headers['Content-Type'] = 'application/json';
		}
		const response = await fetch(url, {
				method: method, mode: 'same-origin', cache: 'no-cache',
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

	async handleBackButton() {
		if (this.backButtonLeavesGroup) {
			await App.fetch("/state", "POST", {action: "leavegroup", index: -1});
			this.showStartScreen();
		} else {
			const stateResp = await App.fetch("/state", "GET", null);
			if (stateResp.activeGroup != -1) {
				this.showGroup(stateResp);
			} else {
				this.showStartScreen();
			}
		}
	}

	setMenu(content) {
		const pagemenu = document.querySelector("#pagemenu");
		const newMenu = pagemenu.cloneNode(false);
		if (content != null) newMenu.appendChild(content);
		pagemenu.parentNode.replaceChild(newMenu, pagemenu);
	}

	setPage(content) {
		const page = document.querySelector("main");
		const newPage = page.cloneNode(false);
		newPage.appendChild(content);
		page.parentNode.replaceChild(newPage, page);
	}

	setTitle(caption, subtitle, backButtonLeavesGroup) {
		this.backButtonLeavesGroup = backButtonLeavesGroup;
		document.querySelector("#title").textContent = caption;
		document.querySelector("#subtitle").textContent = subtitle;
		const backButton = document.querySelector("#back-button");
		const backButtonCaption = backButton.querySelector("#back-button-caption");
		if (backButtonLeavesGroup === null) {
			backButtonCaption.textContent = "";
			backButton.classList.add("empty");
		} else {
			backButtonCaption.textContent = backButtonLeavesGroup ? "Leave" : "Back";
			backButton.classList.remove("empty");
		}
	}

	handlePageMenuClick(controller, item) {
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
			if (controller != null)
				controller.activeMenuEntry = item;
			return true;
		}
	}

	async setGroup(groupIndex) {
		const response = await App.fetch(
			"/state", "POST", {action: "setgroup", index: groupIndex});
		this.showGroup(response);
	}

	async showStartScreen() {
		this.setTitle("Info", null, null);
		this.setMenu(null);
		this.setPage(tmpl.info.view.render(this));
	}

	showGroup(state) {
		if (!Array.isArray(state.modules) ||
				state.modules.length != this.modules.length) {
			throw Error(
					"Invalid response structure (resp.modules not an array or wrong length)");
		} else if (state.activeGroup < 0 ||
				state.activeGroup >= this.groups.length ||state.activeScene < 0 ||
				state.activeScene >= this.groups[state.activeGroup].scenes.length) {
			throw Error("Invalid response (resp.activeScene outside of group scene range)")
		}
		this.activeGroup = state.activeGroup;
		this.setTitle(this.groups[state.activeGroup].name, "", true);
		this.setMenu(this.statePage.genMenu(state.activeScene));
		this.statePage.setSceneData(state.modules);
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
		this.setMenu(this.cfgPage.genMenu(MenuSelect.Active));
		if (this.cfgPage.activeMenuEntry == null) {
			const empty = document.createElement("p");
			empty.textContent = "Select group or system";
			const article = document.createElement("article");
			article.appendChild(empty);
			this.setPage(article);
		} else {
			this.cfgPage.activeMenuEntry.classList.remove("pure-menu-active");
			this.cfgPage.activeMenuEntry.querySelector("a").click();
		}
	}

	async showDatasets() {
		this.setMenu(this.datasetPage.genMenu(MenuSelect.Base));
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
			if (this.stateControllers.hasOwnProperty(module.id)) {
				module.controller = this.stateControllers[module.id];
				this.modules.push(module);
			} else {
				console.error("Missing controller for module \"%s\"", module.id);
			}
		}
		this.fonts = returned.fonts;
		this.textures = returned.textures;
		this.plugins = returned.plugins;
		this.numPluginSystems = returned.numPluginSystems;
		this.messages = returned.messages;
		this.appVersion = returned.appVersion;

		const backButton = document.querySelector("#back-button");
		backButton.addEventListener("click", async e => {
			await this.handleBackButton();
			e.preventDefault();
		});

		if (this.messages != null && this.messages.find(msg => msg.isError)) {
			const mainmenu = document.querySelector("#mainmenu");
			const errorOverlay = document.createElement("div");
			errorOverlay.id = "error-menu-overlay";
			errorOverlay.classList.add("pure-u-1");
			errorOverlay.style.cssText = "width: " + mainmenu.offsetWidth +
					"px; height: " + mainmenu.offsetHeight + "px;";
			errorOverlay.textContent = "Fix errors to enable menu functionality";
			mainmenu.style.position = "relative";
			mainmenu.appendChild(errorOverlay);
			this.showStartScreen();
			return;
		}

		const config = await App.fetch("/data", "GET", null);
		this.systems = config.systems;
		this.groups = config.groups;
		this.activeGroup = -1;
		this.regenGroupListUI();

		this.cfgPage = new DataPage(this, "/config",
				(app, _, url, data) => new ConfigView(app, data, url,
					app.configItemControllers), "Configuration", true);
		this.datasetPage = new DataPage(this, "/data", genDatasetView,
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
				e.target.blur();
				this.showDatasets();
				e.preventDefault();
			});

		this.statePage = new StatePage(this);
		const stateResp = await App.fetch("/state", "GET", null);
		if (stateResp.activeGroup != -1) {
			this.showGroup(stateResp);
		} else {
			this.showStartScreen();
		}
	}
}

const app = new App();