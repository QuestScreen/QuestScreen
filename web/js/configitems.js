tmpl.config.items = {
  selectableFont: new Template("#tmpl-config-selectable-font",
			function (fonts) {
		const families = this.querySelector(".font-families");
		for (let i = 0; i < fonts.length; i++) {
			const option = document.createElement("OPTION");
			option.value = i;
			option.textContent = fonts[i];
			families.appendChild(option);
		}
	}),
	selectableBackground: new Template("#tmpl-config-selectable-background",
			() => {})
}

class SelectableFont {
	constructor(cfg) {
		this.cfg = cfg;
	}

	reset() {
		this.families.value = this.cur.familyIndex;
		this.sizes.value = this.cur.size;
		let style = this.cur.style;
		if (style >= 2) {
			this.styles.querySelector(".italic").classList.add("pure-button-active");
			style -= 2;
		} else {
			this.styles.querySelector(".italic").classList.remove("pure-button-active");
		}
		if (style == 1) {
			this.styles.querySelector(".bold").classList.add("pure-button-active");
		} else {
			this.styles.querySelector(".bold").classList.remove("pure-button-active");
		}
	}

	genUI(app, data, editHandler) {
		this.node = tmpl.config.items.selectableFont.render(app.fonts);
		this.families = this.node.querySelector(".font-families");
		this.sizes = this.node.querySelector(".font-size");
		this.styles = this.node.querySelector(".pure-button-group");

		for (const button of this.styles.querySelectorAll("button")) {
			button.addEventListener("click", () => {
				this.cfg.buttonHandler.bind(this.cfg, button);
				editHandler();
			});
		}
		this.families.addEventListener("change", editHandler);
		this.sizes.addEventListener("change", editHandler);

		if (data == null) {
			this.cur = {
				familyIndex: 0, size: 1, style: 0
			}
		} else {
			this.cur = data;
		}
		this.reset();
		return this.node;
	}

	setEnabled(value) {
		if (value) {
			this.families.disabled = false;
			this.sizes.disabled = false;
			for (const button of this.styles.querySelectorAll("button")) {
				button.disabled = false;
			}
		} else {
			this.families.disabled = true;
			this.sizes.disabled = true;
			for (const button of this.styles.querySelectorAll("button")) {
				button.disabled = true;
			}
		}
	}

	getData() {
		this.cur.familyIndex = parseInt(this.families.value, 10);
		this.cur.size = parseInt(this.sizes.value, 10);
		this.cur.style = 0;
		if (this.styles.querySelector(".bold.pure-button-active") != null) {
			this.cur.style = 1;
		}
		if (this.styles.querySelector(".italic.pure-button-active") != null) {
			this.cur.style += 2;
		}
		return this.cur;
	}
}

class TextureSelector extends DropdownSelector {
	constructor(index, editHandler) {
		super(SelectorKind.atMostOne, true, null);
		this.index = index;
		this.editHandler = editHandler;
	}

	setIndex(index) {
		this.index = index;
		this.setItemSelected(index, true);
	}

	itemClick(index) {
		this.setIndex(index);
		this.editHandler();
	}
}

class SelectableTexturedBackground {
	constructor(cfg) {
		this.cfg = cfg;
	}

	reset() {
		this.primary.value = this.cur.primary;
		this.secondary.value = this.cur.secondary;
		this.textures.setIndex(this.cur.textureIndex);
	}

	genUI(app, data, editHandler) {
		this.node = tmpl.config.items.selectableBackground.render();
		this.primary = this.node.querySelector('input[name=primary]');
		this.secondary = this.node.querySelector('input[name="secondary"]');
		if (data != null) {
			this.cur = data;
		} else {
			this.cur = {primary: "#ffffff", textureIndex: -1, secondary: "#000000"};
		}
		this.textures = new TextureSelector(this.cur.textureIndex, editHandler);
		const texLabel = this.node.querySelector('label[for="texture"]');
		texLabel.parentNode.insertBefore(this.textures.genUi(this.cur.textureIndex,
			app.textures), texLabel.nextSibling);

		this.primary.addEventListener("changed", editHandler);
		this.secondary.addEventListener("changed", editHandler);

		this.reset();
		return this.node;
	}

	setEnabled(value) {
		this.primary.disabled = !value;
		this.secondary.disabled = !value;
		this.textures.setEnabled(value);
	}

	getData() {
		this.cur.primary = this.primary.value;
		this.cur.secondary = this.secondary.value;
		this.cur.textureIndex = this.textures.index;
		return this.cur;
	}
}

app.registerConfigItemController(SelectableFont);
app.registerConfigItemController(SelectableTexturedBackground);