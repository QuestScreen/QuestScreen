tmpl.popup = {
	templateItem: new Template("#tmpl-popup-template-item",
			function(plugin, pluginIndex, tmpl, tmplIndex, ctrl) {
		this.querySelector(".plugin-name").textContent = plugin.name;
		this.querySelector(".template-name").textContent = tmpl.name;
		this.querySelector(".template-descr").textContent = tmpl.description;

		this.querySelector(".pure-menu-link").addEventListener("click", function(e) {
			ctrl.select(this, pluginIndex, tmplIndex);
			e.preventDefault();
		});
		return this.children[0];
	}),
	templateList: new Template("#tmpl-popup-template-list",
			function(plugins, templateSet, ctrl) {
		const list = this.querySelector(".pure-menu-list");
		for (const [pluginIndex, plugin] of plugins.entries()) {
			for (const [tmplIndex, template] of plugin[templateSet].entries()) {
				list.appendChild(tmpl.popup.templateItem.render(
					plugin, pluginIndex, template, tmplIndex, ctrl));
			}
		}
		return this.children[0];
	})
};

class Popup {
	constructor(title, content, confirmCaption, cancelCaption) {
		this.wrapper = document.querySelector("#popup-wrapper");
		this.confirmButton = document.querySelector("#popup-confirm");
		this.cancelButton = document.querySelector("#popup-cancel");
		this.title = title;
		this.content = Array.isArray(content) ? content : [content];
		this.confirmCaption = confirmCaption;
		this.cancelCaption = cancelCaption;
	}

	cleanup(e) {
		this.wrapper.style.display = null;
		this.confirmButton.removeEventListener("click", this.confirmLambda);
		this.cancelButton.removeEventListener("click", this.cancelLambda);
		e.preventDefault();
	}

	confirm(e) {
		if (typeof this.confirmAction === 'function') this.confirmAction();
		else this.resolve(true);
		this.cleanup(e);
	}

	cancel(e) {
		if (typeof this.cancelAction === 'function') this.cancelAction();
		else this.resolve(false);
		this.cleanup(e);
	}

	async show() {
		const popup = this;
		const ret = new Promise(function(resolve, _) {
			popup.resolve = resolve;
		});

		document.querySelector("#popup-title").textContent = this.title;
		const oldContainer = document.querySelector("#popup-content");
		const container = oldContainer.cloneNode(false);
		oldContainer.parentNode.replaceChild(container, oldContainer);
		for (const elm of this.content) {
			container.appendChild(elm);
		}
		this.confirmButton.textContent = this.confirmCaption;
		document.querySelector("#popup").addEventListener(
				"submit", this.confirm.bind(this));
		this.cancelButton.textContent = this.cancelCaption;
		this.cancelButton.addEventListener("click", this.cancel.bind(this));
		if (typeof this.doShow === 'function') {
			this.wrapper.style.visibility = "hidden";
			this.wrapper.style.display = "flex";
			this.doShow();
			// this is required to avoid flickering. I have no idea why.
			// it doesn't work if the timeout simply removes style.visibility.
			this.wrapper.style.display = "none";
			this.wrapper.style.visibility = null;
			setTimeout(() => {this.wrapper.style.display = "flex";}, 10);
		} else {
			this.wrapper.style.display = "flex";
		}

		return await ret;
	}
}

class ConfirmPopup extends Popup {
	constructor(text) {
		super("Confirm", document.createTextNode(text), "OK", "Cancel");
	}
}

class TextInputPopup extends Popup {
	constructor(title, label) {
		const labelE = document.createElement("label");
		labelE.appendChild(document.createTextNode(label));
		const input = document.createElement("input");
		input.type = "text";
		input.required = true;
		super(title, [labelE, input], "OK", "Cancel");
		this.input = input;
	}

	confirmAction() {
		this.resolve(this.input.value);
	}

	cancelAction() {
		this.resolve(null);
	}
}

class TemplateSelectPopup extends Popup {
	constructor(title, plugins, templateSet) {
		const label = document.createElement("label");
		label.appendChild(document.createTextNode("Name: "));
		const nameInput = document.createElement("input");
		nameInput.type = "text";
		nameInput.required = true;
		super(title, [label, nameInput], "OK", "Cancel");
		// must do this after calling super to initialize `this`.
		const tmplList = tmpl.popup.templateList.render(plugins, templateSet, this);
		this.content.push(tmplList);
		this.nameInput = nameInput;
		this.menu = tmplList;
		this.indexName = templateSet.slice(0, -1) + "Index";
	}

	select(item, pluginIndex, templateIndex) {
		if (item.parentNode.classList.contains("pure-menu-active")) {
			if (this.menu.classList.contains("menu-expanded")) {
				this.menu.classList.remove("menu-expanded");
			} else {
				this.menu.classList.add("menu-expanded");
			}
		} else {
			for (const other of this.menu.querySelectorAll(".pure-menu-item")) {
				other.classList.remove("pure-menu-active");
			}
			item.parentNode.classList.add("pure-menu-active");
			this.menu.classList.remove("menu-expanded");
			this.value = {pluginIndex: pluginIndex};
			this.value[this.indexName] = templateIndex;
		}
	}

	doShow() {
		this.menu.classList.add("menu-expanded");
		for (const item of this.menu.querySelectorAll(".template-list .pure-menu-item")) {
			// calculate and explicitly set the height of the item based on the height
			// of the container which can vary due to its variable content.
			// this is required to make our expand/collapse animation work.
			const container = item.querySelector(".template-container");
			// add the .5em vertical padding around the container to the height.
			item.style.height = "calc(" + container.offsetHeight + "px + 1em)";
		}
		// select first item
		this.menu.querySelector(".template-item .pure-menu-link").click();
		this.menu.classList.remove("menu-expanded");
	}

	confirmAction() {
		this.value.name = this.nameInput.value;
		this.resolve(this.value);
	}

	cancelAction() {
		this.resolve(null);
	}
}