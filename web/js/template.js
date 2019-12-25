class Template {
	constructor(id, renderer) {
		this.source = document.querySelector(id);
		if (this.source == null) {
			throw new Error("Template references unknown id \"" + id + "\"!");
		}
		this.renderer = renderer;
	}

	render(...args) {
		return this.renderer.apply(
				document.importNode(this.source, true).content, args);
	}

	static genMenuEntry(name, handler, handlerContext, ...handlerArgs) {
		let item = document.createElement("li");
		item.classList.add("pure-menu-item");
		let a = document.createElement("a");
		item.appendChild(a);
		a.href = "#";
		a.classList.add("pure-menu-link");
		a.textContent = name;
		a.addEventListener('click', handler.bind(
			handlerContext, a, ...handlerArgs));
		return item;
	}
}

let tmpl = {};