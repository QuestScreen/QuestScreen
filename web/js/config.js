const ItemKind = {
	System: 0, Group: 1, Hero: 2
}

tmpl.config = {
	item: new Template("#tmpl-config-item",
			function (itemDesc, app, data) {
		this.querySelector(".config-item-name").textContent =
				itemDesc.config.name;
		const container = this.querySelector("fieldset");
		const checkbox = container.querySelector(".config-item-checkbox");
		const indicator = container.querySelector(".config-edit-indicator");
		const content = itemDesc.handler.genUI(app, data, () => {
			indicator.classList.add("edited");
		});
		if (data == null) {
			itemDesc.handler.setEnabled(false);
		}

		container.querySelector(".config-item-container").appendChild(content);
		checkbox.addEventListener("change", e => {
			itemDesc.enabled = e.currentTarget.checked;
			itemDesc.handler.setEnabled(e.currentTarget.checked);
			indicator.classList.add("edited");
		});
		checkbox.checked = data != null;
	}),
	module: new Template("#tmpl-config-module",
			function(app, moduleDesc, data) {
		const name = this.querySelector(".config-module-name");
		name.textContent = moduleDesc.name;
		const content = this.querySelector(".config-module-content");
		for (let i = 0; i < moduleDesc.items.length; i++) {
			content.appendChild(tmpl.config.item.render(
					moduleDesc.items[i], app, data[i]));
		}
	}),
	view: new Template("#tmpl-config-view",
			function(app, moduleDescs, data, saveHandler) {
		const form = this.querySelector("form");
		const controls = this.querySelector("fieldset");
		for (let i = moduleDescs.length - 1; i >= 0; i--) {
			const desc = moduleDescs[i];
			if (desc != null && desc.items.length > 0) {
				form.insertBefore(tmpl.config.module.render(
					app, desc, data[i]), controls);
			}
		}
		form.addEventListener("submit", e => {
			saveHandler();
			e.preventDefault();
		});
	})
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
	swapButton(button, indicator) {
		if (button.classList.contains("pure-button-active")) {
			button.classList.remove("pure-button-active");
		} else {
			button.classList.add("pure-button-active");
		}
		indicator.classList.add("edited");
	}

	constructor(app, data, url, controllers) {
		this.app = app;
		this.url = url;
		this.controllers = controllers;
		this.buttonHandler = this.swapButton.bind(this);
		this.moduleDescs = [];
		for (let i = 0; i < app.modules.length; i++) {
			if (data[i] == null) {
				this.moduleDescs.push(null);
				continue;
			}
			const moduleItemDescs = [];
			for (let j = 0; j < app.modules[i].config.length; j++) {
				const config = app.modules[i].config[j];
				let handler = null;
				if (this.controllers.hasOwnProperty(config.type)) {
					handler = new this.controllers[config.type](this);
				} else {
					alert("Unknown config item type: " + config.type);
				}

				moduleItemDescs.push(new ModuleItemDesc(
					config, data[i][j] != null, handler));
			}
			this.moduleDescs.push(new ModuleDesc(app.modules[i].name, moduleItemDescs));
		}
	}

	async put() {
		const jsonConfig = [];
		for (const moduleDesc of this.moduleDescs) {
			if (moduleDesc == null) {
				jsonConfig.push(null);
				continue;
			}
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
		await App.fetch(this.url, "PUT", jsonConfig);
		document.querySelector(".config-edit-indicator").classList.remove("edited");
	}

	ui(data) {
		return tmpl.config.view.render(this.app, this.moduleDescs,
			data, this.put.bind(this));
	}
}
