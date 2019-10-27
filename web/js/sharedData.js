var data;

const ItemKind = {
  System: 0,
  Group: 1,
  Hero: 2
}

function setChanged() {
    document.querySelector("#settings-changed").style.visibility = "visible";
}

function swapButton() {
    if (this.classList.contains("pure-button-active")) {
        this.classList.remove("pure-button-active");
    } else {
        this.classList.add("pure-button-active");
    }
    setChanged();
}

function swapSelectableFont() {
    let ui = this.parentNode.parentNode;
    let families = ui.querySelector(".font-families");
    let sizes = ui.querySelector(".font-size");
    let styles = ui.querySelector(".pure-button-group");

    function setValues(family, size, style) {
        families.value = family;
        sizes.value = size;
        if (style >= 2) {
            styles.querySelector(".italic").classList.add("pure-button-active");
            style -= 2;
        } else {
            styles.querySelector(".italic").classList.remove("pure-button-active");
        }
        if (style == 1) {
            styles.querySelector(".bold").classList.add("pure-button-active");
        } else {
            styles.querySelector(".bold").classList.remove("pure-button-active");
        }
    }

    if (this.checked) {
        families.disabled = false;
        sizes.disabled = false;
        styles.querySelectorAll("button").forEach(function (item) {
            item.disabled = false;
        });
        setValues(families.dataset.current, sizes.dataset.current,
            parseInt(styles.dataset.current));
    } else {
        families.dataset.current = families.value;
        sizes.dataset.current = sizes.value;
        styleVal = 0;
        if (styles.querySelector(".bold.pure-button-active") != null) {
            styleVal = 1;
        }
        if (styles.querySelector(".italic.pure-button-active") != null) {
            styleVal += 2;
        }
        styles.dataset.current = styleVal;
        setValues(families.dataset.default, sizes.dataset.default,
            parseInt(styles.dataset.default));
        families.disabled = true;
        sizes.disabled = true;
        styles.querySelectorAll("button").forEach(function (item) {
            item.disabled = true;
        });
    }
}

function postConfig(e) {
    let jsonConfig = [];
    document.querySelector("#main").querySelectorAll(".module-settings-content"
            ).forEach(function (item) {
        let vals = [];
        item.querySelectorAll(".pure-control-group").forEach(function (val) {
            if (val.querySelector(".settings-item-checkbox").checked) {
                let res = {};
                switch(val.dataset.type) {
                    case "SelectableFont":
                        let families = val.querySelector(".font-families");
                        let sizes = val.querySelector(".font-size");
                        let styles = val.querySelector(".pure-button-group");
                        res.familyIndex = parseInt(families.value, 10);
                        res.size = parseInt(sizes.value, 10);
                        res.style = 0;
                        if (styles.querySelector(".bold.pure-button-active") != null) {
                            res.style = 1;
                        }
                        if (styles.querySelector(".italic.pure-button-active") != null) {
                            res.style += 2;
                        }
                        break;
                }
                vals.push(res);
            } else {
                vals.push(null);
            }
        });
        jsonConfig.push(vals);
    });

    return fetch(this.dataset.link, {
        method: 'POST', mode: 'no-cors', cache: 'no-cache', credentials: 'omit',
        headers: { 'Content-Type': 'application/json' },
        redirect: 'error', referrer: 'no-referrer',
        body: JSON.stringify(jsonConfig),
    }).then(function (response) {
        if (response.ok) {
            document.querySelector("#settings-changed").style.visibility = "hidden";
        } else {
            console.log(response);
            alert("Settings update failed!");
        }
    });
}

function genConfigUI(index, settingDef, curValue) {
    let ui = document.importNode(document.querySelector("#tmpl-settings-item"
        ).content, true);
    ui.querySelector(".settings-item-name").textContent = settingDef.name;
    let container = ui.querySelector(".pure-control-group");
    container.dataset.name = settingDef.name;
    container.dataset.type = settingDef.type;
    switch (settingDef.type) {
        case "SelectableFont":
            let fontUI = document.importNode(document.querySelector(
                "#tmpl-settings-selectable-font").content, true);
            let families = fontUI.querySelector(".font-families");
            for (var i = 0; i < data.fonts.length; i++) {
                let option = document.createElement("OPTION");
                option.value = i;
                option.textContent = data.fonts[i];
                families.appendChild(option);
            }
            let sizes = fontUI.querySelector(".font-size");
            let styles = fontUI.querySelector(".pure-button-group");

            families.dataset.default = curValue.default.familyIndex;
            sizes.dataset.default = curValue.default.size;
            styles.dataset.default = curValue.default.style;
            if (curValue.value == null) {
                families.dataset.current = families.dataset.default;
                sizes.dataset.current = sizes.dataset.default;
                styles.dataset.current = styles.dataset.default;
            } else {
                families.dataset.current = curValue.value.familyIndex;
                sizes.dataset.current = curValue.value.size;
                styles.dataset.current = curValue.value.style;
            }

            container.appendChild(fontUI);
            let checkbox = container.querySelector(".settings-item-checkbox");
            checkbox.checked = true;
            if (curValue.value == null) {
                // initialize UI elements to reflect current value, so that they
                // are stored when disabling.
                swapSelectableFont.call(checkbox);
                checkbox.checked = false;
            }
            swapSelectableFont.call(checkbox);
            checkbox.addEventListener("change", swapSelectableFont);
            styles.querySelectorAll("button").forEach(function (item) {
                item.addEventListener("click", swapButton);
            });
            families.addEventListener("change", setChanged);
            sizes.addEventListener("change", setChanged);
            break;
        default:
            let p = document.createElement("P");
            p.textContent = "Unknown setting type: `" + settingDef.type + "`";
            container.appendChild(p);
            break;
    }
    return ui;
}

function getSubject(kind, index) {
    switch (kind) {
        case ItemKind.System: {
            let item = data.systems[index];
            return {
                item: item,
                title: "System „" + item.name + "“",
                link: "/systems/" + item.dirName
            };
        }
        case ItemKind.Group: {
            let item = data.groups[index];
            return {
                item: item,
                title: "Group „" + item.name + "“",
                link: "/groups/" + item.dirName
            };
        }
        default:
            return null;
    }
}

function setMain(node) {
    let main = document.querySelector("#main");
    let newMain = main.cloneNode(false);
    newMain.appendChild(node);
    main.parentNode.replaceChild(newMain, main);
}

function showSettings(e) {
    let subject = getSubject(parseInt(this.dataset.kind), this.dataset.index);
    let heading = subject.title + " Settings";
    let link = "/config" + subject.link;
    fetch(link).then(function (response) {
        if (response.ok) {
            return response.json();
        }
        throw Error("Could not get " + response.url);
    }).then(function (values) {
        let page = document.importNode(document.querySelector(
            "#tmpl-settings-page").content, true);
        page.querySelector("#settings-heading").textContent = heading;
        let moduleTemplate = document.querySelector("#tmpl-settings-module");
        let container = page.querySelector("article");
        let controlSep = container.querySelector("#settings-control-sep");
        for (var i = 0; i < data.modules.length; i++) {
            let mod = data.modules[i];
            let modUI = document.importNode(moduleTemplate.content, true);
            let name = modUI.querySelector("#module-settings-name");
            name.removeAttribute("id");
            name.textContent = mod.name;
            let settings = modUI.querySelector(".module-settings-content");
            settings.dataset.index = i;
            for (var j = 0; j < mod.config.length; j++) {
                settings.appendChild(genConfigUI(j, mod.config[j], values[i][j]));
            }
            container.insertBefore(modUI, controlSep);
        }
        let saveBtn = container.querySelector("#settings-save");
        saveBtn.dataset.link = link;
        saveBtn.addEventListener('click', postConfig);
        setMain(page);
    }).catch(function (error) {
        console.log(error);
    });
}

function updateTitle(e) {
    let value = this.parentNode.querySelector("input").value;
    return fetch("module/title/set", {
        method: 'POST', mode: 'no-cors', cache: 'no-cache', credentials: 'omit',
        headers: { 'Content-Type': 'application/json' },
        redirect: 'error', referrer: 'no-referrer',
        body: JSON.stringify(value),
    }).then(function (response) {
        if (!response.ok) {
            alert("Update failed!");
            console.log(response);
        }
    });
}

function selectBackground(e) {
    let index = parseInt(this.dataset.index, 10);
    let activeName = this.parentNode.parentNode.parentNode.querySelector(".background-state-active");
    let selectedItem = this;
    let oldSelectedItem = this.parentNode.parentNode.querySelector(".pure-menu-selected");
    return fetch("module/background/set", {
        method: 'POST', mode: 'no-cors', cache: 'no-cache', credentials: 'omit',
        headers: {'Content-Type': 'application/json' },
        redirect: 'error', referrer: 'no-referrer',
        body: JSON.stringify(index),
    }).then(function (response) {
        if (response.ok) {
            activeName.textContent = selectedItem.textContent;
            selectedItem.parentNode.classList.add("pure-menu-selected");
            oldSelectedItem.classList.remove("pure-menu-selected");
        } else {
            alert("Update failed!");
            console.log(response);
        }
    });
}

function switchOverlay(e) {
    let adding = this.parentNode.tagName == "LI";
    let index = adding ? parseInt(this.dataset.index, 10) :
        parseInt(this.parentNode.dataset.index, 10);

    let parent = adding ? this.parentNode.parentNode.parentNode.parentNode.parentNode.parentNode :
        this.parentNode.parentNode.parentNode;
    return fetch("module/persons/switch", {
        method: 'POST', mode: 'no-cors', cache: 'no-cache', credentials: 'omit',
        headers: {'Content-Type': 'application/json' },
        redirect: 'error', referrer: 'no-referrer',
        body: JSON.stringify(index),
    }).then(function (response) {
        if (response.ok) {
            parent.querySelector(`.overlay-state-selector li a[data-index=\"${index}\"]`).style.display =
                adding ? "none" : "";
            parent.querySelector(`.visible-overlay-item[data-index=\"${index}\"]`).style.display =
                adding ? "inline-block" : "none";
        } else {
            alert("Update failed!");
            console.log(response);
        }
    });
}

function genMenuEntry(name, index, eventHandler) {
    let item = document.createElement("li");
    item.classList.add("pure-menu-item");
    let a = document.createElement("a");
    item.appendChild(a);
    a.href = "#";
    a.classList.add("pure-menu-link");
    a.textContent = name;
    a.dataset.index = index;
    a.addEventListener('click', eventHandler);
    return item;
}

function selectGroup(e) {
    let groupIndex = this.dataset.index;
    fetch("/groups/" + this.dataset.name).then(function (response) {
        if (response.ok) {
            return response.json();
        }
        throw Error("Could not select group");
    }).then(function (received) {
        if (!Array.isArray(received) || received.length != data.modules.length) {
            throw Error("Invalid response structure (not an array or wrong length");
        }
        let page = document.importNode(document.querySelector(
            "#tmpl-state").content, true);
        page.querySelector("#group-heading").textContent =
            data.groups[groupIndex].name;
        let stateWrapper = page.querySelector("#module-state-wrapper");

        data.modules.forEach(function (module, index) {
            let moduleUI = document.importNode(document.querySelector(
                "#tmpl-state-module").content, true);
            let wrapper = moduleUI.querySelector(".state-module-content");
            moduleUI.querySelector(".state-module-name").textContent =
                module.name;
            switch(module.dirName) {
                case "background":
                    let bgUI = document.importNode(document.querySelector(
                        "#tmpl-background-state").content, true);
                    let selector = bgUI.querySelector(".background-state-selector");
                    let itemList = selector.querySelector("ul");
                    itemList.appendChild(genMenuEntry("None", -1, selectBackground));
                    received[index].items.forEach(function (item, itemIndex) {
                        itemList.appendChild(genMenuEntry(item, itemIndex, selectBackground));
                    });
                    let activeInner = itemList.childNodes[received[index].curIndex + 1];
                    activeInner.classList.add("pure-menu-selected");
                    selector.querySelector(".background-state-active").textContent =
                        activeInner.querySelector("a").textContent;
                    wrapper.appendChild(bgUI);
                    break;
                case "herolist":
                    let heroUI = document.importNode(document.querySelector(
                        "#tmpl-herolist-state").content, true);
                    wrapper.appendChild(heroUI);
                    break;
                case "persons":
                    let personsUI = document.importNode(document.querySelector(
                        "#tmpl-overlay-state").content, true);
                    let visibleContainer = personsUI.querySelector(
                        ".visible-overlays");
                    let selectionContainer = personsUI.querySelector(
                        ".overlay-state-selector ul");
                    received[index].forEach(function (item, index) {
                        let itemUI = document.importNode(document.querySelector(
                            "#tmpl-overlay-item").content, true);
                        let itemDiv = itemUI.querySelector(".visible-overlay-item");
                        itemDiv.querySelector(".overlay-name").textContent =
                            item.name;
                        itemDiv.style.display = item.selected ? "inline-block" : "none";
                        itemDiv.dataset.index = index;
                        itemUI.querySelector(".overlay-close-btn").addEventListener('click', switchOverlay);
                        visibleContainer.appendChild(itemUI);

                        let menuEntry = genMenuEntry(item.name, index, switchOverlay);
                        menuEntry.style.display = item.selected ? "none" : "";
                        selectionContainer.appendChild(menuEntry);
                    });
                    wrapper.appendChild(personsUI);
                    break;
                case "title":
                    let titleUI = document.importNode(document.querySelector(
                        "#tmpl-title-state").content, true);
                    titleUI.querySelector(".title-state-text").value =
                        received[index];
                    titleUI.querySelector(".title-state-text-btn").addEventListener(
                        'click', updateTitle);
                    wrapper.appendChild(titleUI);
                    break;
                default:
                    let msg = document.createElement("p");
                    msg.textContent = "Unknown module: \"" + module.dirName + "\"";
                    wrapper.appendChild(msg);
                    break;
            }
            stateWrapper.appendChild(moduleUI);
        });
        setMain(page);
    }).catch(function (error) {
        console.log(error);
    });
}

document.addEventListener("DOMContentLoaded", function () {
    fetch("/static.json").then(function (response) {
        if (response.ok) {
            return response.json();
        }
        throw Error("Could not get static.json");
    }).then(function (received) {
        let curLast = document.querySelector("#rp-menu-system-heading");
        data = received;

        function appendMenuEntry(template, index, selectedIndex, item, kind) {
            let entry = document.importNode(template.content, true);
            if (index == selectedIndex) {
                entry.querySelector("li").classList.add("pure-menu-selected");
            }
            let link = entry.querySelector("a.pure-menu-link");
            switch (kind) {
                case ItemKind.System:
                    break;
                case ItemKind.Group:
                    link.addEventListener('click', selectGroup);
                    link.dataset.name = item.dirName;
                    link.dataset.index = index;
                    link.href = "#";
                    break;
                case ItemKind.Hero:
                    link.href = "/heroes/" + item.dirName;
                    break;
            }
            link.textContent = item.name;
            let cog = entry.querySelector("a.settings-link");
            cog.addEventListener('click', showSettings);
            cog.dataset.kind = kind;
            cog.dataset.index = index;
            /*cog.dataset.link = "/config" + prefix + item.dirName;
            cog.dataset.name = itemType + " „" + item.name + "“"*/

            curLast.parentNode.insertBefore(entry, curLast.nextSibling);
            curLast = curLast.nextSibling;
        }

        let systemTemplate = document.querySelector("#tmpl-system-entry");
        data.systems.forEach(function (system, index) {
            appendMenuEntry(systemTemplate, index, data.curActiveSystem, system, ItemKind.System);
        });

        let groupTemplate = document.querySelector("#tmpl-group-entry");
        curLast = document.querySelector("#rp-menu-group-heading");
        data.groups.forEach(function (group, index) {
            appendMenuEntry(groupTemplate, index, data.curActiveGroup, group, ItemKind.Group);
        });
        document.querySelector("#rp-menu-list").style.visibility = "visible";
    }).catch(function (error) {
        console.log(error);
    });
});