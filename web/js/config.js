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
		const content = itemDesc.handler.ui(app, data, () => {
			indicator.classList.add("edited");
		});
		if (data == null) {
			itemDesc.handler.setEnabled(false);
		}

		container.querySelector(".config-item-container").appendChild(content);
		checkbox.addEventListener("change", e => {
			itemDesc.setEnabled(e.currentTarget.checked);
			indicator.classList.add("edited");
		});
		checkbox.checked = data != null;
		itemDesc.checkbox = checkbox;
	}),
	module: new Template("#tmpl-config-module",
			function(app, moduleDesc, data) {
		const name = this.querySelector(".config-module-name");
		name.textContent = moduleDesc.name;
		const content = this.querySelector(".config-module-content");
		for (let i = 0; i < moduleDesc.items.length; i++) {
			content.appendChild(moduleDesc.items[i].ui(app, data[i]));
		}
	}),
	view: new Template("#tmpl-config-view", function(controller, data) {
		const form = this.querySelector("form");
		const controls = this.querySelector("fieldset");
		for (let i = controller.moduleDescs.length - 1; i >= 0; i--) {
			const desc = controller.moduleDescs[i];
			if (desc != null && desc.items.length > 0) {
				form.insertBefore(desc.ui(controller.app, data[i]), controls);
			}
		}
		form.addEventListener("submit", e => {
			controller.save();
			e.preventDefault();
		});
		form.addEventListener("reset", e => {
			controller.reset();
			e.preventDefault();
		});
	})
}

class ModuleItemDesc {
	constructor(config, enabled, handler) {
		this.config = config;
		this.enabled = enabled;
		this.origEnabled = enabled;
		this.handler = handler;
	}

	setEnabled(value) {
		this.handler.setEnabled(value);
		this.enabled = value;
	}

	reset() {
		this.handler.reset();
		if (this.origEnabled != this.enabled) {
			this.enabled = this.origEnabled;
			this.handler.setEnabled(this.enabled);
			this.checkbox.checked = this.enabled;
		}
	}

	ui(app, data) {
		return tmpl.config.item.render(this, app, data);
	}
}

class ModuleDesc {
	constructor(name, items) {
		this.name = name;
		this.items = items;
	}

	reset() {
		for (const item of this.items) {
			item.reset();
		}
	}

	ui(app, data) {
		return tmpl.config.module.render(app, this, data);
	}
}

class ConfigView {

	constructor(app, data, url, controllers) {
		this.app = app;
		this.url = url;
		this.controllers = controllers;
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

	removeEditIndicators() {
		for (const indicator of document.querySelectorAll(".config-edit-indicator")) {
			indicator.classList.remove("edited");
		}
	}

	async save() {
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
				itemDesc.origEnabled = itemDesc.enabled;
			}
			jsonConfig.push(vals);
		}
		await App.fetch(this.url, "PUT", jsonConfig);
		this.removeEditIndicators();
	}

	reset() {
		for (const moduleDesc of this.moduleDescs) {
			moduleDesc.reset();
		}
		this.removeEditIndicators();
	}

	ui(data) {
		return tmpl.config.view.render(this, data);
	}
}
