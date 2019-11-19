let tmpl = {
    app: {
        menuGroupEntry: new Template("#tmpl-app-menu-group-entry",
                function(handleSelect, handleSettings) {
            let link = this.querySelector("a.pure-menu-link");
            link.addEventListener('click', handleSelect);
            link.href = "#";
            link.textContent = item.name;
            let cog = entry.querySelector("a.settings-link");
            cog.addEventListener('click', handleSettings);
            return this;
        }),

        menu: new Template("#tmpl-app-menu", function(app) {
            let curLast = this.querySelector("#rp-menu-group-heading");
            for (let [index, group] of app.groups.entries()) {
                let entry = tmpl.app.menuGroupEntry.render(
                    app.selectGroup.bind(app, index),
                    app.groupSettings.bind(app, group)
                );
                curLast.parentNode.insertBefore(entry, curLast.nextSibling);
                curLast = curLast.nextSibling;
            }
            return this;
        }),

        state: {
            module: new Template("#tmpl-app-state-module",
                    function(app, moduleIndex, state) {
                let wrapper = this.querySelector(".state-module-content");
                this.querySelector(".state-module-name").textContent =
                        app.modules[moduleIndex].name;
                wrapper.appendChild(
                    app.controllers[moduleIndex].ui(app, state));
            }),

            page: new Template("#tmpl-app-state-page",
                    function(app, moduleStates) {
                this.querySelector("#group-heading").textContent =
                        app.groups[app.activeGroup].name;
                let stateWrapper = this.querySelector("#module-state-wrapper");
                for (let [index, state] of moduleStates.entries()) {
                    stateWrapper.appendChild(
                        tmpl.app.state.module.render(app, index, state));
                }
            })
        }
    }
};

class App {
    constructor() {
        this.controllers = {};
        this.modules = [];
        this.systems = [];
        this.fonts = [];
    }

    /* registers a controller. must be called before init(). */
    registerController(controller) {
        this.controllers[controller.id] = controller;
    }

    static fetch = async function(url, method, content) {
        let body = content == null ? null : JSON.stringify(content);
        let headers = { 'X-Clacks-Overhead': 'GNU Terry Pratchett'};
        if (body != null) {
            headers['Content-Type'] = 'application/json';
        }
        let response = await fetch(url, {
            method: method, mode: 'no-cors', cache: 'no-cache',
            credentials: 'omit', redirect: 'follow', referrer: 'no-referrer',
            headers: headers, body: body,
        });
        if (response.ok) {
            return await response.json();
        } else {
            throw new Error("failed to fetch " + url);
        }
    }

    setMain(content) {
        let main = document.querySelector("#main");
        let newMain = main.cloneNode(false);
        newMain.appendChild(content);
        main.parentNode.replaceChild(newMain, main);
        initDropdowns();
    }

    async selectGroup(index) {
        let moduleStates = await App.fetch("/groups/" + this.groups[index]);
        if (!Array.isArray(moduleStates) ||
            moduleStates.length != this.modules.length) {
            throw Error(
                "Invalid response structure (not an array or wrong length");
        }
        this.activeGroup = index;
        let page = tmpl.app.state.page.render(this, moduleStates);
        this.setMain(page);
        for(let [index, entry] of
                document.querySelectorAll(".rp-menu-group-entry")) {
            if (index == this.activeGroup) {
                entry.classList.add("pure-menu-selected");
            } else {
                entry.classList.remove("pure-menu-selected");
            }
        }
    }

    groupSettings(group) {
        
    }

    /* queries the global config from the server and initializes the app. */
    async init() {
        let returned = await App.fetch("/app", "GET", null);
        for (const [key, value] of Object.entries(returned.modules)) {
            if (this.controllers.hasOwnProperty(key)) {
                value.controller = this.controllers[key];
                this.modules.push(value);
            } else {
                console.error("Missing controller for module \"%s\"", key);
            }
        }
        this.systems = returned.systems;
        this.groups = returned.groups;
        this.fonts = returned.fonts;
        this.activeGroup = returned.activeGroup;

        let menu = document.querySelector("#menu");
        let renderedMenu = menu.cloneNode(false);
        renderedMenu.appendChild(tmpl.app.menu.render(this));
        
        if (this.activeGroup != -1) {
            this.selectGroup(this.groups[this.activeGroup]);
        }
    }
}

let app = new App();