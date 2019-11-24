
class Background extends ListSelector {
	constructor() {
		super(SelectorKind.atMostOne, true, null);
		this.id = "background";
	}

	ui(app, state) {
		return this.genListUi(state.curIndex, state.items);
	}

	async listItemClick(index) {
		await App.fetch("module/background/set", "POST", index);
		this.setListItemSelected(index, true);
	}
}

tmpl.herolist = {
	state: new Template("#tmpl-base-herolist-state",
			function (state, ctrl, listUI) {
		let allSwitch = this.querySelector(".herolist-switch-all");
		if (!state.global) {
			allSwitch.textContent = "Show All";
		} else {
			allSwitch.classList.add("pure-button-primary");
		}
		allSwitch.addEventListener('click', ctrl.swapAll.bind(ctrl, allSwitch));
		this.querySelector(".herolist-container").appendChild(listUI);
		return this;
	})
}

class HeroList extends ListSelector {
	constructor() {
		super(SelectorKind.multiple, false, "Heroes");
		this.id = "herolist";
	}

	ui(app, state) {
		let captions = app.groups[app.activeGroup].heroes.map(h => h.name);
		let listUI = this.genListUi(state.heroes, captions);
		return tmpl.herolist.state.render(state, this, listUI);
	}

	async listItemClick(index) {
		let shown = await App.fetch(
			"module/herolist/switchHero", "POST",  index);
		this.setListItemSelected(index, shown);
	}

	async swapAll(node) {
		let shown = await App.fetch(
			"module/herolist/switchGlobal", "POST", null);
		if (shown) {
			node.classList.add("pure-button-primary");
			node.textContent = "Hide All";
		} else {
			node.classList.remove("pure-button-primary");
			node.textContent = "Show All";
		}
	}
}

class Overlay extends ListSelector {
	constructor() {
		super(SelectorKind.multiple, true, "Overlay");
		this.id = "overlay";
	}

	ui(app, state) {
		let captions = state.map(s => s.name);
		let visible = state.map(s => s.selected);
		return this.genListUi(visible, captions);
	}

	async listItemClick(index) {
		let visible = await App.fetch("module/overlay/switch", "POST", index);
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
		let input = node.parentNode.querySelector("input");
		if (node.classList.contains("pure-button-primary")) {
			value = input.value;
		}
		let newValue = await App.fetch("module/title/set", "POST", value);
		input.value = newValue;
	}
}

app.registerController(new Background());
app.registerController(new HeroList());
app.registerController(new Overlay());
app.registerController(new Title());