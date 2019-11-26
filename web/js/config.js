const ItemKind = {
	System: 0, Group: 1, Hero: 2
}

tmpl.config = {
	item: new Template("#tmpl-config-item",
			function (itemDesc, content, checked) {
		this.querySelector(".settings-item-name").textContent =
				itemDesc.config.name;
		let container = this.querySelector(".pure-control-group");
		container.appendChild(content);
		let checkbox = container.querySelector(".settings-item-checkbox");
		checkbox.addEventListener("change", function() {
			itemDesc.enabled = this.checked;
			itemDesc.handler.setEnabled(this.checked);
		});
		checkbox.checked = checked;
		return this;
	}),
	module: new Template("#tmpl-config-module",
			function(app, moduleDesc, data) {
		let name = this.querySelector(".settings-module-name");
		name.textContent = moduleDesc.name;
		let settings = this.querySelector(".module-settings-content");
		for (let i = 0; i < moduleDesc.items.length; i++) {
			settings.appendChild(tmpl.config.item.render(
					moduleDesc.items[i],
					moduleDesc.items[i].handler.genUI(app, data[i]),
					data != null));
		}
		return this;
	}),
	page: new Template("#tmpl-config-page",
			function(app, heading, moduleDescs, data, saveHandler) {
		this.querySelector("#settings-heading").textContent = heading;
		let container = this.querySelector("article");
		let controlSep = this.querySelector("#settings-control-sep");
		for (let i = 0; i < moduleDescs.length; i++) {
			container.insertBefore(tmpl.config.module.render(
				app, moduleDescs[i], data[i]), controlSep);
		}
		container.querySelector("#settings-save").addEventListener("click",
			saveHandler);
		return this;
	}),
	selectableFont: new Template("#tmpl-config-selectable-font",
			function (fonts) {
		let families = this.querySelector(".font-families");
		for (let i = 0; i < fonts.length; i++) {
			let option = document.createElement("OPTION");
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

class ConfigPage {
	setChanged() {
		document.querySelector("#settings-changed").style.visibility = "visible";
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
			let moduleItemDescs = [];
			for (let j = 0; j < app.modules[i].config.length; j++) {
				let config = app.modules[i].config[j];
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
		let jsonConfig = [];
		for (const moduleDesc of this.moduleDescs) {
			let vals = [];
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
		document.querySelector("#settings-changed").style.visibility = "hidden";
	}

	ui(heading, data) {
		return tmpl.config.page.render(this.app, heading, this.moduleDescs,
			data, this.post.bind(this));
	}
}