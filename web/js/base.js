tmpl.background = {
    state: new Template("#tmpl-base-background-state",
            function(state, ctrl) {
        let selector = this.querySelector(".background-state-selector");
        let itemList = selector.querySelector("ul");
        itemList.appendChild(
            Template.genMenuEntry("None", ctrl.select, ctrl, -1));
        for (let [index, item] of state.items.entries()) {
            itemList.appendChild(
                Template.genMenuEntry(item, ctrl.select, ctrl, itemIndex));
        }
        let activeInner = itemList.childNodes[state.curIndex + 1];
        activeInner.classList.add("pure-menu-selected");
        selector.querySelector(".background-state-active").textContent =
            activeInner.querySelector("a").textContent;
        return this;
    })
}

class Background {
    constructor() {
        this.id = "background";
    }

    ui(app, state) {
        return tmpl.background.state.render(state, this);
    }

    async select(node, index) {
        let activeName = node.parentNode.parentNode.parentNode.querySelector(
            ".background-state-active");
        let oldSelectedItem = node.parentNode.parentNode.querySelector(".pure-menu-selected");
        await App.fetch("module/background/set", "POST", JSON.stringify(index));
        activeName.textContent = node.textContent;
        node.parentNode.classList.add("pure-menu-selected");
        oldSelectedItem.classList.remove("pure-menu-selected");
    }
}

tmpl.herolist = {
    stateItem: new Template("#tmpl-base-herolist-item",
            function (app, ctrl, index, selected) {
        if (selected) {
            this.querySelector("li").classList.add("pure-menu-selected");
        }
        this.querySelector(".herolist-item-name").textContent = 
            app.groups[app.activeGroup].heroes[index].name;
        let a = this.querySelector("a");
        a.addEventListener('click', ctrl.swapSingle.bind(ctrl, a, index));
    }),
    state: new Template("#tmpl-base-herolist-state",
            function (app, state, ctrl) {
        let allSwitch = this.querySelector(".herolist-switch-all");
        if (!state.global) {
            allSwitch.textContent = "Show All";
        } else {
            allSwitch.classList.add("pure-button-primary");
        }
        allSwitch.addEventListener('click', ctrl.swapAll.bind(ctrl, allSwitch));
        let itemContainer = this.querySelector(".herolist-selector ul");
        for (let [index, selected] of state.heroes.entries()) {
            itemContainer.appendChild(tmpl.herolist.stateItem.render(
                app, this, index, selected));
        }
        return this;
    })
}

class HeroList {
    constructor() {
        this.id = "herolist";
    }

    ui(app, state) {
        return tmpl.herolist.state.render(app, state, this);
    }

    async swapSingle(node, index) {
        let li = node.parentNode;
        let shown = await App.fetch(
            "module/herolist/switchHero", "POST",  index);
        if (shown) {
            li.classList.add("pure-menu-selected");
        } else {
            li.classList.remove("pure-menu-selected");
        }
    }

    async swapAll(node) {
        let shown = await App.fetch(
            "module/herolist/switchGlobal", "POST", null);
        if (shown) {
            node.classList.add("pure-button-primary");
            node.textContent = "Hide All";
        } else {
            node.classList.remove("pure-button-primary");
            node.textContent = "Show All";
        }
    }
}

tmpl.overlay = {
    stateItem: new Template("#tmpl-base-overlay-state-item",
            function(ctrl, item, index) {
        let itemDiv = this.querySelector(".visible-overlay-item");
        itemDiv.querySelector(".overlay-name").textContent =
            item.name;
        itemDiv.style.display = item.selected ? "inline-block" : "none";
        let btn = this.querySelector(".overlay-close-btn");
        btn.addEventListener(
            'click', ctrl.swap.bind(ctrl, btn, index));
        return this;
    }),
    state: new Template("#tmpl-base-overlay-state",
            function(state, ctrl) {
        let visibleContainer = this.querySelector(
            ".visible-overlays");
        let selectionContainer = this.querySelector(
            ".overlay-state-selector ul");
        for (let [index, item] of state.entries()) {
            visibleContainer.appendChild(tmpl.overlay.stateItem.render(
                ctrl, item, index));
            let menuEntry = Template.genMenuEntry(
                item.name, index, ctrl.swap, ctrl, index);
            menuEntry.style.display = item.selected ? "none" : "";
            selectionContainer.appendChild(menuEntry);
        }
        return this;
    })
}

class Overlay {
    constructor() {
        this.id = "overlay";
    }

    ui(app, state) {
        return tmpl.overlay.state.render(state, this);
    }

    async swap(node, index) {
        let adding = node.parentNode.tagName == "LI";
    
        let parent = adding  ?
            node.parentNode.parentNode.parentNode.parentNode.parentNode.parentNode :
            node.parentNode.parentNode.parentNode;
        let visible = await App.fetch("module/overlay/switch", "POST", index);
        parent.querySelector(
            `.overlay-state-selector li a[data-index=\"${index}\"]`).style.display =
                    visible ? "none" : "";
            parent.querySelector(`.visible-overlay-item[data-index=\"${index}\"]`).style.display =
                    visible ? "inline-block" : "none";
    }
}

tmpl.title = {
    state: new Template("#tmpl-base-title-state",
            function(value, ctrl) {
        this.querySelector(".title-state-text").value = value;
        for (const button of this.querySelectorAll(".title-state-text-btn")) {
            button.addEventListener('click', ctrl.update.bind(ctrl, button));
        }
    })
}

class Title {
    constructor() {
        this.id = "title";
    }

    ui(app, state) {
        return tmpl.title.state.render(state, this);
    }

    async update(node) {
        let value = "";
        let input = node.parentNode.querySelector("input");
        if (node.classList.contains("pure-button-primary")) {
            value = input.value;
        }
        let newValue = await App.fetch("module/title/set", "POST", value);
        input.value = newValue;
    }
}

app.registerController(new Background());
app.registerController(new HeroList());
app.registerController(new Overlay());
app.registerController(new Title());