const datasets = {
	genList: function(list, items, ctrl, delHandler, addHandler, firstRemovable) {
		const adder = list.querySelector(".data-list-add");
		for (const [index, item] of items.entries()) {
			list.insertBefore(tmpl.data.listItem.render(
				index, item, ctrl, delHandler, firstRemovable <= index), adder);
		}
		adder.querySelector(".pure-menu-link").addEventListener("click", e => {
			addHandler.call(ctrl);
			e.preventDefault();
		});
	}
}

tmpl.data = {
	listItem: new Template("#tmpl-data-list-item",
			function(index, item, ctrl, handler, removable) {
		this.querySelector("span").textContent = item.name;
		const link = this.querySelector("a");
		if (removable) {
			link.href = "#";
			link.addEventListener("click", e => {
				handler.call(ctrl, index);
				e.preventDefault();
			});
			link.classList.add("enabled");
			link.querySelector("i").classList.add("fa-minus-square");
		} else {
			link.querySelector("i").classList.add("fa-cubes");
		}
	}),
	base: new Template("#tmpl-data-base", function(app, ctrl) {
		datasets.genList(this.querySelector(".data-system-list"), app.systems,
				ctrl, ctrl.delSystem, ctrl.createSystem, app.numPluginSystems);
		datasets.genList(this.querySelector(".data-group-list"), app.groups,
				ctrl, ctrl.delGroup, ctrl.createGroup, 0);
	}),
	system: new Template("#tmpl-data-system", function(ctrl, system) {
		const form = this.children[0];
		form.querySelector("#system-name").value = system.name;
		form.addEventListener("submit", function (e) {
			ctrl.save.call(ctrl, this);
			e.preventDefault();
		});
		form.querySelector("button.revert").addEventListener("click", e => {
			form.querySelector("#system-name").value = system.name;
			e.preventDefault();
		});
		return form;
	}),
	group: new Template("#tmpl-data-group", function(ctrl, group) {
		this.querySelector("#group-name").value = group.name;
		this.querySelector("form").addEventListener("submit", function (e) {
			ctrl.save.call(ctrl, this);
			e.preventDefault();
		});
		this.querySelector("button.revert").addEventListener("click", e => {
			form.querySelector("#group-name").value = group.name;
		});
		datasets.genList(this.querySelector(".data-scene-list"), group.scenes,
				ctrl, ctrl.delScene, ctrl.createScene, 1);
		datasets.genList(this.querySelector(".data-hero-list"), group.heroes,
				ctrl, ctrl.delHero, ctrl.createHero, 0);
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
			this.app.setMenu(this.app.datasetPage.genMenu());
			this.app.setPage(this.ui());
		}
	}

	async createSystem() {
		const popup = new TextInputPopup("Create system", "Name:");
		const request = await popup.show();
		if (request !== null) {
			const data = await App.fetch("data/systems", "POST", request);
			this.app.systems = data;
			this.app.setMenu(this.app.datasetPage.genMenu());
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
			this.app.setMenu(this.app.datasetPage.genMenu());
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
			this.app.setMenu(this.app.datasetPage.genMenu());
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
		this.app.setMenu(this.app.datasetPage.genMenu());
		this.app.setPage(this.ui());
		this.app.datasetPage.updateTitle(this.system);
	}

	ui() {
		return tmpl.data.system.render(this, this.system);
	}
}

class GroupDataView {
	constructor(app, url, group) {
		this.app = app;
		this.url = url;
		this.group = group;
	}

	async save(form) {
		this.app.groups = await App.fetch(this.url, "PUT",
				{name: form.querySelector("#group-name").value});
		for (const g of this.app.groups) {
			if (g.id == this.group.id) {
				this.group = g;
				break;
			}
		}
		this.app.setMenu(this.app.datasetPage.genMenu());
		this.app.setPage(this.ui());
		this.app.datasetPage.updateTitle(this.group);
	}

	async createScene() {
		const popup = new TemplateSelectPopup("Create a scene", this.app.plugins,
				"sceneTemplates");
		const request = await popup.show();
		if (request !== null) {
			const data = await App.fetch(this.url + "/scenes", "POST", request);
			this.group.scenes = data;
			this.app.regenGroupListUI();
			this.app.setMenu(this.app.datasetPage.genMenu());
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
			this.app.setMenu(this.app.datasetPage.genMenu());
			this.app.setPage(this.ui());
		}
	}

	async createHero() {
		alert("TODO");
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

	ui() {
		return tmpl.data.group.render(this, this.group);
	}
}

class SceneDataView {
	constructor(app, url, scene) {
		this.app = app;
		this.url = url;
		this.scene = scene;
	}
}

function genDatasetView(app, kind, url, item) {
	switch (kind) {
		case datakind.Base: return new BaseDataView(app, url);
		case datakind.System: return new SystemDataView(app, url, item);
		case datakind.Group: return new GroupDataView(app, url, item);
		case datakind.Scene: return new SceneDataView(app, url, item);
	}
}