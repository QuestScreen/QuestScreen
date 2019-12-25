const ItemKind = {
	System: 0, Group: 1, Hero: 2
}

tmpl.config = {
	menu: new Template("#tmpl-app-config-menu",
			function(app, configPage) {
		configPage.activeMenuEntry = null;
		const list = this.querySelector(".pure-menu-list");

		let curLast = list.querySelector(".config-menu-system-heading");
		for (const [index, system] of app.systems.entries()) {
			const entry = tmpl.app.pageMenuEntry.render(
				app, configPage.viewSystem.bind(configPage, system), system.name);
			// Safari doesn't support firstElementChild on DocumentFragment
			curLast = curLast.parentNode.insertBefore(entry, curLast.nextSibling);
			if (app.activeGroup == -1 && index == 0) {
				configPage.activeMenuEntry = entry;
			}
		}

		curLast = this.querySelector(".config-menu-group-heading");
		for (const [index, group] of app.groups.entries()) {
			const entry = tmpl.app.pageMenuEntry.render(
					app, configPage.viewGroup.bind(configPage, group), group.name);
			curLast = curLast.parentNode.insertBefore(entry, curLast.nextSibling);
			if (index == app.activeGroup) {
				configPage.activeMenuEntry = entry;
			}
		}
		return this;
	}),
	item: new Template("#tmpl-config-item",
			function (itemDesc, content, checked) {
		this.querySelector(".config-item-name").textContent =
				itemDesc.config.name;
		const container = this.querySelector("fieldset");
		container.appendChild(content);
		const checkbox = container.querySelector(".config-item-checkbox");
		checkbox.addEventListener("change", e => {
			itemDesc.enabled = e.currentTarget.checked;
			itemDesc.handler.setEnabled(e.currentTarget.checked);
			itemDesc.handler.cfg.setChanged();
		});
		checkbox.checked = checked;
		return this;
	}),
	module: new Template("#tmpl-config-module",
			function(app, moduleDesc, data) {
		const name = this.querySelector(".config-module-name");
		name.textContent = moduleDesc.name;
		const content = this.querySelector(".module-config-content");
		for (let i = 0; i < moduleDesc.items.length; i++) {
			content.appendChild(tmpl.config.item.render(
					moduleDesc.items[i],
					moduleDesc.items[i].handler.genUI(app, data[i]),
					data[i] != null));
		}
		return this;
	}),
	view: new Template("#tmpl-config-view",
			function(app, moduleDescs, data, saveHandler) {
		const container = this.querySelector("article");
		for (let i = moduleDescs.length - 1; i >= 0; i--) {
			if (moduleDescs[i].items.length > 0) {
				container.insertBefore(tmpl.config.module.render(
					app, moduleDescs[i], data[i]), container.childNodes[0]);
			}
		}
		container.querySelector(".config-save").addEventListener("click", e => {
			saveHandler();
			e.preventDefault();
		});
		return this;
	}),
	selectableFont: new Template("#tmpl-config-selectable-font",
			function (fonts) {
		const families = this.querySelector(".font-families");
		for (let i = 0; i < fonts.length; i++) {
			const option = document.createElement("OPTION");
			option.value = i;
			option.textContent = fonts[i];
			families.appendChild(option);
		}
		return this;
	})
}

class SelectableFont {
	constructor(cfg) {
		this.cfg = cfg;
	}

	setValues() {
		this.families.value = this.cur.familyIndex;
		this.sizes.value = this.cur.size;
		let style = this.cur.style;
		if (style >= 2) {
			this.styles.querySelector(".italic").classList.add("pure-button-active");
			style -= 2;
		} else {
			this.styles.querySelector(".italic").classList.remove("pure-button-active");
		}
		if (style == 1) {
			this.styles.querySelector(".bold").classList.add("pure-button-active");
		} else {
			this.styles.querySelector(".bold").classList.remove("pure-button-active");
		}
	}

	genUI(app, data) {
		this.node = tmpl.config.selectableFont.render(app.fonts);
		this.families = this.node.querySelector(".font-families");
		this.sizes = this.node.querySelector(".font-size");
		this.styles = this.node.querySelector(".pure-button-group");

		for (const button of this.styles.querySelectorAll("button")) {
			button.addEventListener("click", this.cfg.buttonHandler.bind(null, button));
		}
		this.families.addEventListener("change", this.cfg.changedHandler);
		this.sizes.addEventListener("change", this.cfg.changedHandler);

		if (data == null) {
			this.cur = {
				familyIndex: 0, size: 1, style: 0
			}
			this.setEnabled(false);
		} else {
			this.cur = data;
		}
		this.setValues();
		return this.node;
	}

	setEnabled(value) {
		if (value) {
			this.families.disabled = false;
			this.sizes.disabled = false;
			for (const button of this.styles.querySelectorAll("button")) {
				button.disabled = false;
			}
		} else {
			this.families.disabled = true;
			this.sizes.disabled = true;
			for (const button of this.styles.querySelectorAll("button")) {
				button.disabled = true;
			}
		}
	}

	getData() {
		this.cur.familyIndex = parseInt(this.families.value, 10);
		this.cur.size = parseInt(this.sizes.value, 10);
		this.cur.style = 0;
		if (this.styles.querySelector(".bold.pure-button-active") != null) {
			this.cur.style = 1;
		}
		if (this.styles.querySelector(".italic.pure-button-active") != null) {
			this.cur.style += 2;
		}
		return this.cur;
	}
}

class ModuleItemDesc {
	constructor(config, enabled, handler) {
		this.config = config;
		this.enabled = enabled;
		this.handler = handler;
	}
}

class ModuleDesc {
	constructor(name, items) {
		this.name = name;
		this.items = items;
	}
}

class ConfigView {
	setChanged() {
		document.querySelector(".config-changed").style.visibility = "visible";
	}

	swapButton(button) {
		if (button.classList.contains("pure-button-active")) {
			button.classList.remove("pure-button-active");
		} else {
			button.classList.add("pure-button-active");
		}
		this.setChanged();
	}

	constructor(app, data, url) {
		this.app = app;
		this.url = url;
		this.changedHandler = this.setChanged.bind(this);
		this.buttonHandler = this.swapButton.bind(this);
		this.moduleDescs = [];
		for (let i = 0; i < app.modules.length; i++) {
			const moduleItemDescs = [];
			for (let j = 0; j < app.modules[i].config.length; j++) {
				const config = app.modules[i].config[j];
				let handler = null;
				switch (config.type) {
					case "SelectableFont":
						handler = new SelectableFont(this);
				}

				moduleItemDescs.push(new ModuleItemDesc(
					config, data[i][j] != null, handler));
			}
			this.moduleDescs.push(new ModuleDesc(app.modules[i].name, moduleItemDescs));
		}
	}

	async post() {
		const jsonConfig = [];
		for (const moduleDesc of this.moduleDescs) {
			const vals = [];
			for (const itemDesc of moduleDesc.items) {
				if (itemDesc.enabled) {
					vals.push(itemDesc.handler.getData());
				} else {
					vals.push(null);
				}
			}
			jsonConfig.push(vals);
		}
		await App.fetch(this.url, "POST", jsonConfig);
		document.querySelector(".config-changed").style.visibility = "hidden";
	}

	ui(data) {
		return tmpl.config.view.render(this.app, this.moduleDescs,
			data, this.post.bind(this));
	}
}

class ConfigPage {
	constructor(app) {
		this.app = app;
	}

	async loadView(url, title, subtitle) {
		const cfgData = await App.fetch(url, "GET", null);
		const view = new ConfigView(this.app, cfgData, url);
		this.app.setPage(view.ui(cfgData));
		this.app.setTitle(title, subtitle);
	}

	async viewSystem(system) {
		this.loadView("/config/systems/" + system.id,
									"System Configuration", system.name);
	}

	async viewGroup(group) {
		this.loadView("/config/groups/" + group.id,
									"Group Configuration", group.name);
	}

	genMenu() {
		return tmpl.config.menu.render(app, this);
	}
}