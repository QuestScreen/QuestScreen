tmpl.app = {
	groupEntry: new Template("#tmpl-app-group-entry",
			function(app, name, groupIndex) {
		let link = this.querySelector(".pure-menu-link");
		link.textContent = name;
		link.addEventListener('click', e => {
			const groupMenuBtn = document.querySelector("#open-group-menu");
			groupMenuBtn.click();
			groupMenuBtn.blur();
			app.showState(groupIndex);
		});
		// return directly the <li> element.
		return this.children[0];
	}),
	pageMenuEntry: new Template("#tmpl-app-page-menu-entry",
			function(app, handler, name) {
		let link = this.querySelector(".pure-menu-link");
		link.textContent = name;
		link.addEventListener("click", e => {
			app.setActivePageMenuItem(link.parentNode);
			handler();
		});
		return this.children[0];
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

	setMenu(content) {
		let pagemenu = document.querySelector("#pagemenu");
		let newMenu = pagemenu.cloneNode(false);
		newMenu.appendChild(content);
		pagemenu.parentNode.replaceChild(newMenu, pagemenu);
	}

	setPage(content) {
		const page = document.querySelector("main");
		const newPage = page.cloneNode(false);
		newPage.appendChild(content);
		page.parentNode.replaceChild(newPage, page);
	}

	setTitle(caption) {
		document.querySelector("#title").textContent = caption;
	}

	setActivePageMenuItem(item) {
		const pagemenu = document.querySelector("#pagemenu");
		for (const item of pagemenu.querySelectorAll(".pure-menu-item")) {
			item.classList.remove("pure-menu-active");
		}
		item.classList.add("pure-menu-active");
		pagemenu.classList.remove("pagemenu-expanded");
	}

	async showState(groupIndex) {
		const moduleStates = await App.fetch("/groups/" + this.groups[groupIndex].id);
		if (!Array.isArray(moduleStates) ||
			moduleStates.length != this.modules.length) {
			throw Error(
					"Invalid response structure (not an array or wrong length");
		}
		this.activeGroup = groupIndex;
		const page = tmpl.app.state.page.render(this, moduleStates);
		this.setPage(page);
		this.setTitle(app.groups[groupIndex].name);
		this.setMenu(document.createTextNode("TODO"));
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
			this.showState(this.activeGroup);
		}
		this.cfgPage = new ConfigPage(this);
		document.querySelector("#show-config").addEventListener(
			"click", e => {
				e.target.blur();
				this.showConfig();
			});
		document.querySelector("#header-toggle").addEventListener(
				"click", e => { this.toggleHeader(e.currentTarget); });
	}
}

let app = new App();