class Template {
	constructor(id, renderer) {
		this.source = document.querySelector(id);
		if (this.source == null) {
			throw new Error("Template references unknown id \"" + id + "\"!");
		}
		this.renderer = renderer;
	}

	render(...args) {
		const node = document.importNode(this.source, true).content;
		const ret = this.renderer.apply(node, args);
		return ret === undefined ? node : ret;
	}

	static genMenuEntry(name, handler, handlerContext, ...handlerArgs) {
		const item = document.createElement("li");
		item.classList.add("pure-menu-item");
		const a = document.createElement("a");
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