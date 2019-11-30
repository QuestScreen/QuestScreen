const ACTIVE_CLASS_NAME = "pure-menu-active",
      DISMISS_EVENT     = (window.hasOwnProperty("ontouchstart")) ?
	                        "touchstart" : "mousedown";

class DropdownHandler {
	constructor(parent) {
		this.parent = parent;
		this.closed = true;
		this.menu = parent.querySelector(".pure-menu-children");
		this.link = parent.querySelector(".pure-menu-link");
		this.link.addEventListener("click", this.toggle.bind(this));
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
		} else {
			this.parent.classList.remove(ACTIVE_CLASS_NAME);
			this.link.focus();
		}
		this.closed = !this.closed;
	}
}