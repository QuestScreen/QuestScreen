const datasets = {
	genList: function(list, items, ctrl, delHandler, addHandler) {
		const adder = list.querySelector(".data-list-add");
		for (const [index, item] of items.entries()) {
			list.insertBefore(tmpl.data.listItem.render(
				index, item, ctrl, delHandler), adder);
		}
		adder.querySelector(".pure-menu-link").addEventListener("click", e => {
			addHandler.call(ctrl);
			e.preventDefault();
		});
	}
}

tmpl.data = {
	listItem: new Template("#tmpl-data-list-item", function(index, item, ctrl, handler) {
		this.querySelector("span").textContent = item.name;
		this.querySelector("button").addEventListener("click", e => {
			handler.call(ctrl, index);
			e.preventDefault();
		});
	}),
	base: new Template("#tmpl-data-base", function(app, ctrl) {
		datasets.genList(this.querySelector(".data-system-list"), app.systems,
				ctrl, ctrl.delSystem, ctrl.createSystem);
		datasets.genList(this.querySelector(".data-group-list"), app.groups,
				ctrl, ctrl.delGroup, ctrl.createGroup);
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
			await App.fetch("/datasets/system/delete", "POST", system.id);
			this.app.systems.splice(index, 1);
			this.app.setMenu(this.app.datasetPage.genMenu());
		}
	}

	async createSystem() {
		const popup = new TextInputPopup("Create system", "Name:");
		const request = await popup.show();
		if (request !== null) {
			const data = await App.fetch("/datasets/system/create", "POST", request);
			this.app.systems = data.systems;
			this.app.setMenu(this.app.datasetPage.genMenu());
		}
	}

	async delGroup(index) {
		const group = this.app.groups[index];
		const popup = new ConfirmPopup("Really delete group " + group.name + "?");
		if (await popup.show()) {
			await App.fetch("/datasets/group/delete", "POST", group.id);
			this.app.groups.splice(index, 1);
			this.app.regenGroupListUI();
			this.app.setMenu(this.app.datasetPage.genMenu());
		}
	}

	async createGroup() {
		const popup = new TemplateSelectPopup("Create a group", this.app.plugins,
				"groupTemplates");
		const request = await popup.show();
		if (request !== null) {
			const data = await App.fetch("/datasets/group/create", "POST", request);
			this.app.groups = data.groups;
			this.app.regenGroupListUI();
			this.app.setMenu(this.app.datasetPage.genMenu());
		}
	}

	ui() {
		return tmpl.data.base.render(this.app, this);
	}
}

class SystemDataView {
	constructor(app, url) {
		this.app = app;
		this.url = url;
	}
}

class GroupDataView {
	constructor(app, url) {
		this.app = app;
		this.url = url;
	}
}

class SceneDataView {
	constructor(app, url) {
		this.app = app;
		this.url = url;
	}
}


function genDatasetView(app, kind, url) {
	switch (kind) {
		case datakind.Base: return new BaseDataView(app, url);
		case datakind.System: return new SystemDataView(app, url);
		case datakind.Group: return new GroupDataView(app, url);
		case datakind.Scene: return new SceneDataView(app, url);
	}
}