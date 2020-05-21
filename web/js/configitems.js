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

	ui(app, data, editHandler) {
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
		this.decodeData(this.primary, this.cur.primary);
		this.decodeData(this.secondary, this.cur.secondary);
		this.textures.setIndex(this.cur.textureIndex);
	}

	ui(app, data, editHandler) {
		this.node = tmpl.config.items.selectableBackground.render();
		this.primary = {
			color: this.node.querySelector('input[name="primary-color"]'),
			opacity: this.node.querySelector('input[name="primary-opacity"]')
		};
		this.secondary = {
			color: this.node.querySelector('input[name="secondary-color"]'),
			opacity: this.node.querySelector('input[name="secondary-opacity"]')
		};
		if (data != null) {
			this.cur = data;
		} else {
			this.cur = {primary: "#ffffffff", textureIndex: -1, secondary: "#000000ff"};
		}
		this.textures = new TextureSelector(this.cur.textureIndex, editHandler);
		const texLabel = this.node.querySelector('label[for="texture"]');
		texLabel.parentNode.insertBefore(this.textures.ui(this.cur.textureIndex,
			app.textures), texLabel.nextSibling);

		this.primary.color.addEventListener("change", editHandler);
		this.primary.opacity.addEventListener("change", editHandler);
		this.secondary.color.addEventListener("change", editHandler);
		this.secondary.opacity.addEventListener("change", editHandler);

		this.reset();
		return this.node;
	}

	setEnabled(value) {
		this.primary.disabled = !value;
		this.secondary.disabled = !value;
		this.textures.setEnabled(value);
	}

	encodeData(set) {
		let aHex = Number(set.opacity.value).toString(16);
		if (aHex.length == 1) aHex = "0" + aHex;
		return set.color.value + aHex;
	}

	decodeData(set, value) {
		set.color.value = value.substring(0, 7);
		set.opacity.value = parseInt(value.substring(7), 16);
	}

	getData() {
		this.cur.primary = this.encodeData(this.primary);
		this.cur.secondary = this.encodeData(this.secondary);
		this.cur.textureIndex = this.textures.index;
		return this.cur;
	}
}

app.registerConfigItemController("core:Font", SelectableFont);
app.registerConfigItemController("core:Background", SelectableTexturedBackground);