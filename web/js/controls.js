const ACTIVE_CLASS_NAME = "pure-menu-active",
      DISMISS_EVENT     = (window.hasOwnProperty("ontouchstart")) ?
	                        "touchstart" : "mousedown";

const smartphoneMode = window.matchMedia("screen and (max-width: 35.5em)");

const SelectorKind = Object.freeze({
	atMostOne: Symbol("atMostOne"),
	exactlyOne: Symbol("exactlyOne"),
	multiple: Symbol("multiple")
});

// this class implements UI behavior and is used by the dropdown template.
// to create a new UI element, subclass DropdownSelector.
class DropdownHandler {
	constructor(parent) {
		this.parent = parent;
		this.closed = true;
		this.menu = parent.querySelector(".pure-menu-children");
		this.link = parent.querySelector(".pure-menu-link");
		this.link.addEventListener("click", e => {
			if (!this.link.classList.contains("pure-menu-disabled")) {
				this.toggle();
			}
			e.preventDefault();
		});
		document.addEventListener(DISMISS_EVENT, e => {
			if (e.target !== this.link && !this.menu.contains(e.target)) {
				this.hide();
			}
		});
	}

	hide() {
		if (!this.closed) this.toggle();
	}

	toggle() {
		if (this.closed) {
			this.parent.classList.add(ACTIVE_CLASS_NAME);
			if (smartphoneMode.matches)
				this.menu.style.height = (this.menu.children.length * 2) + "em";
		} else {
			this.parent.classList.remove(ACTIVE_CLASS_NAME);
			if (smartphoneMode.matches) this.menu.style.height = "";
			this.link.blur();
		}
		this.closed = !this.closed;
	}
}

tmpl.controls = {
	switch: new Template("#tmpl-controls-switch", function(name, handler, active) {
		const input = this.querySelector("input");
		input.name = name;
		input.checked = active;
		if (handler !== null) {
			input.addEventListener("change", function() {
				handler(this);
			});
		}
		return this.children[0];
	}),
	dropdown: {
		visible: new Template("#tmpl-controls-dropdown-visible", function() {}),
		invisible: new Template("#tmpl-controls-dropdown-invisible", function() {}),
		item: new Template("#tmpl-controls-dropdown-item",
				function(ctrl, index, visible, caption, handler) {
			const name = this.querySelector(".dropdown-item-name");
			const status = ctrl.showVisible ? tmpl.controls.dropdown.visible :
					tmpl.controls.dropdown.invisible;
			const a = name.parentNode;
			a.insertBefore(status.render(), name);
			name.textContent = caption;
			if (visible) {
				this.querySelector("li").classList.add("pure-menu-selected");
			}
			a.addEventListener("click", e => {
				ctrl.itemClick(index);
				if (ctrl.kind != SelectorKind.multiple) {
					handler.hide();
				}
				e.preventDefault();
			});
		}),
		root: new Template("#tmpl-controls-dropdown",
				function(ctrl, visible, captions) {
			const children = this.querySelector(".pure-menu-children");
			const handler = new DropdownHandler(children.parentNode);
			if (ctrl.kind == SelectorKind.atMostOne) {
				children.appendChild(tmpl.controls.dropdown.item.render(
					ctrl, -1, visible == -1, "None", handler));
			}
			for (const [index, item] of captions.entries()) {
				const itemVisible = ctrl.kind == SelectorKind.multiple ?
						visible[index] : visible == index;
				children.appendChild(tmpl.controls.dropdown.item.render(
						ctrl, index, itemVisible, item, handler));
			}

			const menuCaption = this.querySelector(".dropdown-caption");
			if (ctrl.kind == SelectorKind.multiple) {
				menuCaption.textContent = ctrl.menuName;
			} else if (visible == -1) {
				menuCaption.textContent = "None";
			} else {
				menuCaption.textContent = captions[visible];
			}
		})
	},
};

// base class for a dropdown UI element
class DropdownSelector {
	// menuName only relevant for kind == SelectorKind.multiple
	constructor(kind, showVisible, menuName) {
		this.kind = kind;
		this.showVisible = showVisible;
		this.menuName = menuName;
	}

	ui(visible, captions) {
		const ret = tmpl.controls.dropdown.root.render(this, visible, captions);
		this.uiItems = ret.querySelector(".pure-menu-children").children;
		if (this.kind != SelectorKind.multiple) {
			this.menuCaption = ret.querySelector(".dropdown-caption");
		}
		return ret;
	}

	setItemSelected(index, selected) {
		if (this.kind == SelectorKind.multiple) {
			const item = this.uiItems[index];
			if (selected) {
				item.classList.add("pure-menu-selected");
			} else {
				item.classList.remove("pure-menu-selected");
			}
		} else {
			const actualIndex =
					this.kind == SelectorKind.atMostOne ? index + 1 : index;
			// for … of might not work for older browsers on HTMLCollection.
			for (let itemIndex = 0; itemIndex < this.uiItems.length; itemIndex++) {
				const item = this.uiItems[itemIndex];
				if (actualIndex == itemIndex) {
					item.classList.add("pure-menu-selected");
					this.menuCaption.textContent =
							item.querySelector(".dropdown-item-name").textContent;
				} else {
					item.classList.remove("pure-menu-selected");
				}
			}
		}
	}

	setEnabled(value) {
		if (value) {
			this.menuCaption.parentNode.classList.remove("pure-menu-disabled");
		} else {
			this.menuCaption.parentNode.classList.add("pure-menu-disabled");
		}
	}

	async itemClick(index) {
		// override this, finish by updating UI via setItemSelected
		throw new Error("Missing itemClick implementation!");
	}
}