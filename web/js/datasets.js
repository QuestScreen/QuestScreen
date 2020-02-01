const datasets = {
	genList: function(list, items, ctrl, delHandler, addHandler, firstRemovable) {
		const adder = list.querySelector(".data-list-link");
		for (const [index, item] of items.entries()) {
			list.insertBefore(tmpl.data.listItem.render(
				index, item, ctrl, delHandler, firstRemovable <= index), adder);
		}
		adder.addEventListener("click", e => {
			addHandler.call(ctrl);
			e.preventDefault();
		});
	},
	setEdited: function() {
		this.parentNode.classList.add("edited");
	}
}

tmpl.data = {
	listItem: new Template("#tmpl-data-list-item",
			function(index, item, ctrl, handler, removable) {
		this.querySelector("span").textContent = item.name;
		const link = this.querySelector("a");
		if (removable) {
			link.addEventListener("click", e => {
				handler.call(ctrl, index);
				e.preventDefault();
			});
			link.classList.add("delete");
			link.querySelector("i").classList.add("fa-minus-square");
		} else {
			link.querySelector("i").classList.add("fa-cubes");
		}
	}),
	general: new Template("#tmpl-data-general", function(
			itemName, ctrl, additionalUI) {
		const id = itemName + "-name";
		const form = this.querySelector("form");
		const input = form.querySelector("input");
		input.id = id;
		input.value = ctrl[itemName].name;
		input.addEventListener("input", datasets.setEdited);
		form.querySelector("label").htmlFor = id;
		if (additionalUI) {
			const fieldset = form.querySelector("fieldset");
			const controls = fieldset.querySelector(".pure-controls");
			fieldset.insertBefore(additionalUI, controls);
		}
		form.addEventListener("submit", function (e) {
			ctrl.save.call(ctrl, this);
			for (const controlGroup of form.querySelectorAll(".pure-control-group")) {
				controlGroup.classList.remove("edited");
			}
			e.preventDefault();
		});
		form.addEventListener("reset", e => {
			input.value = ctrl[itemName].name;
			if (ctrl.revert) {
				ctrl.revert.call(ctrl, additionalUI);
			}
			for (const controlGroup of form.querySelectorAll(".pure-control-group")) {
				controlGroup.classList.remove("edited");
			}
			e.preventDefault();
		});
		return this.children[0];
	}),
	base: new Template("#tmpl-data-base", function(app, ctrl) {
		datasets.genList(this.querySelector(".systems"), app.systems,
				ctrl, ctrl.delSystem, ctrl.createSystem, app.numPluginSystems);
		datasets.genList(this.querySelector(".groups"), app.groups,
				ctrl, ctrl.delGroup, ctrl.createGroup, 0);
	}),
	system: new Template("#tmpl-data-system", function(ctrl, system) {
		const article = this.children[0];
		article.appendChild(tmpl.data.general.render("system", ctrl,
				null));
		return article;
	}),
	group: {
		systemSelector: new Template("#tmpl-data-group-system-selector",
				function(ctrl) {
			const controlGroup = this.children[0];
			const systemSelect = ctrl.systemSelector.ui(ctrl.app, () => {
				controlGroup.classList.add("edited");
			});
			const label = controlGroup.querySelector("label");
			controlGroup.insertBefore(systemSelect, label.nextSibling);
			return controlGroup;
		}),
		hero: new Template("#tmpl-data-group-hero", function(ctrl, hero, index) {
			const sheet = this.children[0];
			const form = sheet.querySelector("form");
			const submit = form.querySelector("button.pure-button-primary");
			const cancel = form.querySelector("button.revert");
			const nameInput = form.querySelector('input[name="name"]');
			const descrInput = form.querySelector('input[name="description"]');
			const deleteButton = sheet.querySelector(".data-sheet-delete");
			if (hero) {
				nameInput.value = hero.name;
				descrInput.value = hero.description;
				nameInput.addEventListener("input", datasets.setEdited);
				descrInput.addEventListener("input", datasets.setEdited);
				deleteButton.addEventListener(
						"click", e => {
					ctrl.delHero.call(ctrl, index);
					e.preventDefault();
				});
				form.addEventListener("reset", e => {
					nameInput.value = hero.name;
					descrInput.value = hero.description;
					nameInput.parentNode.classList.remove("edited");
					descrInput.parentNode.classList.remove("edited");
					e.preventDefault();
				});
			} else {
				sheet.classList.add("create");
				sheet.addEventListener("click", () => {
					sheet.classList.add("active");
				});
				form.querySelector(".data-sheet-header").textContent = "Create Hero";
				submit.textContent = "Create";
				cancel.textContent = "Cancel";
				form.addEventListener("reset", e => {
					nameInput.value = "";
					descrInput.value = "";
					sheet.classList.remove("active");
					e.preventDefault();
				});
				sheet.removeChild(deleteButton);
			}
			form.addEventListener("submit", e => {
				const data = {name: nameInput.value, description: descrInput.value};
				ctrl.submitHeroData(data, hero);
				e.preventDefault();
			});
			return sheet;
		}),
		view: new Template("#tmpl-data-group", function(ctrl, group) {
			const article = this.children[0];
			article.insertBefore(tmpl.data.general.render("group", ctrl,
					tmpl.data.group.systemSelector.render(ctrl)), article.firstChild);
			datasets.genList(this.querySelector(".scenes"), group.scenes,
					ctrl, ctrl.delScene, ctrl.createScene, 1);
			for (const [index, hero] of group.heroes.entries()) {
				article.appendChild(tmpl.data.group.hero.render(ctrl, hero, index));
			}
			article.appendChild(tmpl.data.group.hero.render(ctrl, null, -1));
			return article;
		})
	},
	sceneModule: new Template("#tmpl-data-scene-module",
			function(app, ctrl, module, index) {
		this.querySelector(".plugin-name").textContent =
				app.plugins[module.pluginIndex].name;
		this.querySelector(".module-name").textContent = module.name;
		this.querySelector(".module-toggle").appendChild(tmpl.controls.switch.render(
				module.id, ctrl.toggle.bind(ctrl, index), ctrl.scene.modules[index]));
		return this.children[0];
	}),
	sceneModules: new Template("#tmpl-data-scene-modules", function(app, ctrl) {
		const list = this.querySelector(".modules");
		for (const [index, module] of app.modules.entries()) {
			list.appendChild(tmpl.data.sceneModule.render(
				app, ctrl, module, index));
		}
		return this.children[0];
	}),
	scene: new Template("#tmpl-data-scene", function(app, ctrl, scene) {
		const article = this.querySelector("article");
		article.insertBefore(tmpl.data.general.render("scene", ctrl,
				tmpl.data.sceneModules.render(app, ctrl)), article.firstChild);
		return article;
	})
}

class BaseDataView {
	constructor(app, url) {
		this.app = app;
		this.url = url;
	}

	async delSystem(index) {
		const system = this.app.systems[index];
		const popup = new ConfirmPopup("Really delete system " + system.name + "?");
		if (await popup.show()) {
			await App.fetch("data/systems/" + system.id, "DELETE", null);
			this.app.systems.splice(index, 1);
			this.app.setMenu(this.app.datasetPage.genMenu(MenuSelect.Previous));
			this.app.setPage(this.ui());
		}
	}

	async createSystem() {
		const popup = new TextInputPopup("Create system", "Name:");
		const request = await popup.show();
		if (request !== null) {
			const data = await App.fetch("data/systems", "POST", request);
			this.app.systems = data;
			this.app.setMenu(this.app.datasetPage.genMenu(MenuSelect.Previous));
			this.app.setPage(this.ui());
		}
	}

	async delGroup(index) {
		const group = this.app.groups[index];
		const popup = new ConfirmPopup("Really delete group " + group.name + "?");
		if (await popup.show()) {
			await App.fetch("data/groups/" + group.id, "DELETE", null);
			this.app.groups.splice(index, 1);
			this.app.regenGroupListUI();
			this.app.setMenu(this.app.datasetPage.genMenu(MenuSelect.Previous));
			this.app.setPage(this.ui());
		}
	}

	async createGroup() {
		const popup = new TemplateSelectPopup("Create a group", this.app.plugins,
				"groupTemplates");
		const request = await popup.show();
		if (request !== null) {
			const data = await App.fetch("data/groups", "POST", request);
			this.app.groups = data;
			this.app.regenGroupListUI();
			this.app.setMenu(this.app.datasetPage.genMenu(MenuSelect.Previous));
			this.app.setPage(this.ui());
		}
	}

	ui() {
		return tmpl.data.base.render(this.app, this);
	}
}

class SystemDataView {
	constructor(app, url, system) {
		this.app = app;
		this.url = url;
		this.system = system;
	}

	async save(form) {
		this.app.systems = await App.fetch(this.url, "PUT",
				{name: form.querySelector("#system-name").value});
		for (const s of this.app.systems) {
			if (s.id == this.system.id) {
				this.system = s;
				break;
			}
		}
		this.app.setMenu(this.app.datasetPage.genMenu(MenuSelect.Previous));
		this.app.setPage(this.ui());
		this.app.datasetPage.updateTitle(this.system);
	}

	ui() {
		return tmpl.data.system.render(this, this.system);
	}
}

class SystemSelector extends DropdownSelector {
	constructor(curIndex) {
		super(SelectorKind.atMostOne, true, null);
		this.curIndex = curIndex;
		this.originalIndex = curIndex;
	}

	ui(app, changeHandler) {
		const captions = app.systems.map(s => s.name);
		this.changeHandler = changeHandler;
		return this.genUi(this.curIndex, captions);
	}

	async itemClick(index) {
		this.curIndex = index;
		this.setItemSelected(index, true);
		this.changeHandler();
	}
}

class GroupDataView {
	constructor(app, url, group) {
		this.app = app;
		this.url = url;
		this.group = group;
		this.systemSelector = new SystemSelector(group.systemIndex);
	}

	async save(form) {
		this.app.groups = await App.fetch(this.url, "PUT",
				{name: form.querySelector("#group-name").value,
				 systemIndex: this.systemSelector.curIndex});
		this.systemSelector.originalIndex = this.systemSelector.curIndex;
		for (const g of this.app.groups) {
			if (g.id == this.group.id) {
				this.group = g;
				break;
			}
		}
		this.app.setMenu(this.app.datasetPage.genMenu(MenuSelect.Previous));
		this.app.setPage(this.ui());
		this.app.datasetPage.updateTitle(this.group);
	}

	revert() {
		this.systemSelector.itemClick(this.systemSelector.originalIndex);
	}

	async createScene() {
		const popup = new TemplateSelectPopup("Create a scene", this.app.plugins,
				"sceneTemplates");
		const request = await popup.show();
		if (request !== null) {
			const data = await App.fetch(this.url + "/scenes", "POST", request);
			this.group.scenes = data;
			this.app.regenGroupListUI();
			this.app.setMenu(this.app.datasetPage.genMenu(MenuSelect.Previous));
			this.app.setPage(this.ui());
		}
	}

	async delScene(index) {
		const scene = this.group.scenes[index];
		const popup = new ConfirmPopup("Really delete scene " + scene.name + "?");
		if (await popup.show()) {
			await App.fetch(this.url + "/scenes/" + scene.id, "DELETE", null);
			this.group.scenes.splice(index, 1);
			this.app.regenGroupListUI();
			this.app.setMenu(this.app.datasetPage.genMenu(MenuSelect.Previous));
			this.app.setPage(this.ui());
		}
	}

	async delHero(index) {
		const hero = this.group.heroes[index];
		const popup = new ConfirmPopup("Really delete hero " + hero.name + "?");
		if (await popup.show()) {
			await App.fetch(this.url + "/heroes/" + hero.id, "DELETE", null);
			this.group.heroes.splice(index, 1);
			this.app.setPage(this.ui());
		}
	}

	async submitHeroData(data, hero) {
		const url = this.url + (hero == null ? "/heroes" : "/heroes/" + hero.id);
		this.group.heroes = await App.fetch(url, hero == null ? "POST" : "PUT", data);
		this.app.setPage(this.ui());
	}

	ui() {
		return tmpl.data.group.view.render(this, this.group);
	}
}

class SceneDataView {
	constructor(app, url, group, scene) {
		this.app = app;
		this.url = url;
		this.scene = scene;
		this.group = group;
		this.curModules = [...scene.modules];
	}

	async save(form) {
		this.group.scenes = await App.fetch(this.url, "PUT",
				{name: form.querySelector("#scene-name").value,
			   modules: this.curModules});
		for (const s of this.group.scenes) {
			if (s.id == this.scene.id) {
				this.scene = s;
				break;
			}
		}
		for (const item of form.querySelectorAll(".data-list-item")) {
			item.classList.remove("edited");
		}
		this.curModules = [...this.scene.modules];
		this.app.setMenu(this.app.datasetPage.genMenu(MenuSelect.Previous));
		this.app.setPage(this.ui());
		this.app.datasetPage.updateTitle({name: this.group.name + ' ' + this.scene.name});
	}

	revert(ui) {
		this.curModules = [...this.scene.modules];
		for (const [index, input] of ui.querySelectorAll("label.switch > input").entries()) {
			input.checked = this.curModules[index];
			input.parentNode.parentNode.parentNode.classList.remove("edited");
		}
	}

	async toggle(index, input) {
		this.curModules[index] = input.checked;
		input.parentNode.parentNode.parentNode.classList.add("edited");
	}

	ui() {
		return tmpl.data.scene.render(this.app, this, this.scene);
	}
}

function genDatasetView(app, kind, url, item) {
	switch (kind) {
		case datakind.Base: return new BaseDataView(app, url);
		case datakind.System: return new SystemDataView(app, url, item);
		case datakind.Group: return new GroupDataView(app, url, item);
		case datakind.Scene:
			return new SceneDataView(app, url, item.group, item.scene);
	}
}