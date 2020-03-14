tmpl.info = {
	modulePath: new Template("#tmpl-info-module-path", function(plugin, module) {
		this.querySelector(".plugin-name").textContent = plugin;
		this.querySelector(".module-name").textContent = module;
	}),
	module: new Template("#tmpl-info-module", function(module, pluginName) {
		this.querySelector(".module-path").appendChild(
				tmpl.info.modulePath.render(pluginName, module.name));
		this.querySelector(".module-id").textContent = module.id;
		return this.children[0];
	}),
	coreMessage: new Template("#tmpl-info-core-message", function(msg) {
		const tr = this.children[0];
		tr.querySelector("i").classList.add(msg.isError ?
			"fa-times-circle" : "fa-exclamation-triangle");
		tr.querySelector(".text").textContent = msg.text;
		tr.classList.add(msg.isError ? "error" : "warning");
		return tr;
	}),
	moduleMessage: new Template("#tmpl-info-module-message",
			function(plugin, msg) {
		const tr = this.children[0];
		tr.querySelector("i").classList.add(msg.isError ?
			"fa-exclamation-triangle" : "fa-times-circle");
		tr.querySelector(".module-path").appendChild(
				tmpl.info.modulePath.render(plugin, msg.moduleIndex));
		tr.querySelector(".text").textContent = msg.text;
		tr.classList.add(msg.isError ? "error" : "warning");
		return this.children[0];
	}),
	messages: new Template("#tmpl-info-messages",
			function(app) {
		const messageData = this.querySelector("tbody");
		for(const message of app.messages) {
			if (message.moduleIndex == -1) {
				messageData.appendChild(tmpl.info.coreMessage.render(message));
			} else {
				const module = app.modules[message.moduleIndex];
				messageData.appendChild(tmpl.info.moduleMessage.render(
					app.plugins[module.pluginIndex].name, message));
			}
		}
		return this.children[0];
	}),
	view: new Template("#tmpl-info", function(app) {
		this.querySelector(".app-version").textContent = app.appVersion;
		const moduleList = this.querySelector("tbody");
		for (const module of app.modules) {
			moduleList.appendChild(tmpl.info.module.render(
				module, app.plugins[module.pluginIndex].name));
		}
		if (app.messages != null && app.messages.length > 0) {
			this.appendChild(tmpl.info.messages.render(app));
		}
	})
}