const ACTIVE_CLASS_NAME = "pure-menu-active",
      DISMISS_EVENT     = (window.hasOwnProperty("ontouchstart")) ?
	                        "touchstart" : "mousedown";

const smartphoneMode = window.matchMedia("screen and (max-width: 35.5em)");

class DropdownHandler {
	constructor(parent) {
		this.parent = parent;
		this.closed = true;
		this.menu = parent.querySelector(".pure-menu-children");
		this.link = parent.querySelector(".pure-menu-link");
		this.link.addEventListener("click", e => {
			this.toggle();
			e.preventDefault();
		});
		document.addEventListener(DISMISS_EVENT, e => {
			if (e.target !== this.link && !this.menu.contains(e.target)) {
				this.hide();
			}
			e.preventDefault();
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

		}
		this.closed = !this.closed;
	}
}