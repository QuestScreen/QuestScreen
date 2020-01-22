
class Background extends ListSelector {
	constructor() {
		super(SelectorKind.atMostOne, true, null);
		this.id = "background";
	}

	ui(app, state) {
		return this.genListUi(state.curIndex, state.items);
	}

	async listItemClick(index) {
		await App.fetch("state/background", "PUT", index);
		this.setListItemSelected(index, true);
	}
}

tmpl.herolist = {
	state: new Template("#tmpl-base-herolist-state",
			function (state, ctrl, listUI) {
		const allSwitch = this.querySelector(".herolist-switch-all");
		if (!state.global) {
			allSwitch.textContent = "Show All";
		} else {
			allSwitch.classList.add("pure-button-primary");
		}
		allSwitch.addEventListener('click', ctrl.swapAll.bind(ctrl, allSwitch));
		this.querySelector(".herolist-container").appendChild(listUI);
	})
}

class HeroList extends ListSelector {
	constructor() {
		super(SelectorKind.multiple, false, "Heroes");
		this.id = "herolist";
	}

	ui(app, state) {
		const captions = app.groups[app.activeGroup].heroes.map(h => h.name);
		const listUI = this.genListUi(state.heroes, captions);
		return tmpl.herolist.state.render(state, this, listUI);
	}

	async listItemClick(index) {
		const shown = await App.fetch(
			"state/herolist/" + app.groups[app.activeGroup].heroes[index].id, "PUT",
			!this.uiItems[index].classList.contains("pure-menu-selected"));
		this.setListItemSelected(index, shown);
	}

	async swapAll(node) {
		const shown = await App.fetch(
			"state/herolist", "PUT", !node.classList.contains("pure-button-primary"));
		if (shown) {
			node.classList.add("pure-button-primary");
			node.textContent = "Hide All";
		} else {
			node.classList.remove("pure-button-primary");
			node.textContent = "Show All";
		}
	}
}

class Overlays extends ListSelector {
	constructor() {
		super(SelectorKind.multiple, true, "Overlays");
		this.id = "overlays";
	}

	ui(app, state) {
		const captions = state.map(s => s.name);
		const visible = state.map(s => s.selected);
		return this.genListUi(visible, captions);
	}

	async listItemClick(index) {
		const visible = await App.fetch("state/overlays", "PUT",
				{resourceIndex: index, visible:
					!this.uiItems[index].classList.contains("pure-menu-selected")});
		this.setListItemSelected(index, visible);
	}
}

tmpl.title = {
	state: new Template("#tmpl-base-title-state",
			function(value, ctrl) {
		this.querySelector(".title-state-text").value = value;
		for (const button of this.querySelectorAll(".title-state-text-btn")) {
			button.addEventListener('click', ctrl.update.bind(ctrl, button));
		}
		return this;
	})
}

class Title {
	constructor() {
		this.id = "title";
	}

	ui(app, state) {
		return tmpl.title.state.render(state, this);
	}

	async update(node) {
		let value = "";
		const input = node.parentNode.querySelector("input");
		if (node.classList.contains("pure-button-primary")) {
			value = input.value;
		}
		const newValue = await App.fetch("state/title", "PUT", value);
		input.value = newValue;
	}
}

app.registerController(new Background());
app.registerController(new HeroList());
app.registerController(new Overlays());
app.registerController(new Title());