const SelectorKind = Object.freeze({
	atMostOne: Symbol("atMostOne"),
	exactlyOne: Symbol("exactlyOne"),
	multiple: Symbol("multiple")
});

tmpl.state = {
	list: {
		visible: new Template("#tmpl-state-list-visible",
				function() {
			return this;
		}),
		invisible: new Template("#tmpl-state-list-invisible",
				function() {
			return this;
		}),
		item: new Template("#tmpl-state-list-item",
				function(ctrl, index, visible, caption) {
			let name = this.querySelector(".state-list-item-name");
			let status = ctrl.showVisible ? tmpl.state.list.visible :
					tmpl.state.list.invisible;
			let a = name.parentNode;
			a.insertBefore(status.render(), name);
			name.textContent = caption;
			if (visible) {
				this.querySelector("li").classList.add("pure-menu-selected");
			}
			a.addEventListener("click", ctrl.listItemClick.bind(ctrl, index));
			return this;
		}),
		root: new Template("#tmpl-state-list",
				function(ctrl, visible, captions) {
			let children = this.querySelector(".pure-menu-children");
			if (ctrl.kind == SelectorKind.atMostOne) {
				children.appendChild(tmpl.state.list.item.render(
					ctrl, -1, visible == -1, "None"));
			}
			for (const [index, item] of captions.entries()) {
				const itemVisible = ctrl.kind == SelectorKind.multiple ?
						visible[index] : visible == index;
				children.appendChild(tmpl.state.list.item.render(
						ctrl, index, itemVisible, item));
			}
			new DropdownHandler(children.parentNode);

			let menuCaption = this.querySelector(".state-list-caption");
			if (ctrl.kind == SelectorKind.multiple) {
				menuCaption.textContent = ctrl.menuName;
			} else if (visible == -1) {
				menuCaption.textContent = "None";
			} else {
				menuCaption.textContent = captions[visible];
			}
			return this;
		})
	}
}

class ListSelector {
	// menuName only relevant for kind == SelectorKind.multiple
	constructor(kind, showVisible, menuName) {
		this.kind = kind;
		this.showVisible = showVisible;
		this.menuName = menuName;
	}

	genListUi(visible, captions) {
		let ret = tmpl.state.list.root.render(this, visible, captions);
		this.uiItems = ret.querySelector(".pure-menu-children").children;
		if (this.kind != SelectorKind.multiple) {
			this.menuCaption = ret.querySelector(".state-list-caption");
		}
		return ret;
	}

	setListItemSelected(index, selected) {
		if (this.kind == SelectorKind.multiple) {
			let item = this.uiItems.children[index];
			if (selected) {
				item.classList.add("pure-menu-selected");
			} else {
				item.classList.remove("pure-menu-selected");
			}
		} else {
			let actualIndex = this.kind == SelectorKind.atMostOne ? index + 1 : index;
			// for … of might not work for older browsers on HTMLCollection.
			for (let itemIndex = 0; itemIndex < this.uiItems.length; itemIndex++) {
				let item = this.uiItems[itemIndex];
				if (actualIndex == itemIndex) {
					item.classList.add("pure-menu-selected");
					this.menuCaption.textContent =
							item.querySelector(".state-list-item-name").textContent;
				} else {
					item.classList.remove("pure-menu-selected");
				}
			}
		}
	}

	async listItemClick(index) {
		// override this, finish by updating UI via setListItemSelected
		throw new Error("Missing listItemClick implementation!");
	}
}