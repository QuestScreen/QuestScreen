var fonts;

function swapButton() {
    if (this.classList.contains("pure-button-active")) {
        this.classList.remove("pure-button-active");
    } else {
        this.classList.add("pure-button-active");
    }
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

function genConfigUI(name, data) {
    let ui = document.importNode(document.querySelector("#tmpl-settings-item").content, true);
    ui.querySelector(".settings-item-name").textContent = name;
    let container = ui.querySelector(".pure-control-group");
    switch (data.Type) {
        case "SelectableFont":
            let fontUI = document.importNode(document.querySelector("#tmpl-settings-selectable-font").content, true);
            let families = fontUI.querySelector(".font-families");
            for (var i = 0; i < fonts.length; i++) {
                let option = document.createElement("OPTION");
                option.value = i;
                option.textContent = fonts[i];
                families.appendChild(option);
            }
            let sizes = fontUI.querySelector(".font-size");
            let styles = fontUI.querySelector(".pure-button-group");

            families.dataset.default = data.Default.FamilyIndex;
            sizes.dataset.default = data.Default.Size;
            styles.dataset.default = data.Default.Style;
            if (data.Value == null) {
                families.dataset.current = families.dataset.default;
                sizes.dataset.current = sizes.dataset.default;
                styles.dataset.current = styles.dataset.default;
            } else {
                families.dataset.current = data.Value.FamilyIndex;
                sizes.dataset.current = data.Value.Size;
                styles.dataset.current = data.Value.Style;
            }

            container.appendChild(fontUI);
            let checkbox = container.querySelector(".settings-item-checkbox");
            checkbox.checked = true;
            if (data.Value == null) {
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
            break;
        default:
            let p = document.createElement("P");
            p.textContent = "Unknown setting type: `" + data.Type + "`";
            container.appendChild(p);
            break;
    }
    return ui;
}

function showSettings(e) {
    let heading = this.dataset.name + " Settings";
    fetch(this.dataset.link).then(function (response) {
        if (response.ok) {
            return response.json();
        }
        throw Error("Could not get " + response.url);
    }).then(function (data) {
        let page = document.importNode(document.querySelector("#tmpl-settings-page").content, true);
        page.querySelector("#settings-heading").textContent = heading;
        let moduleTemplate = document.querySelector("#tmpl-settings-module");
        let container = page.querySelector("article");
        for (var modName in data) {
            let modUI = document.importNode(moduleTemplate.content, true);
            let name = modUI.querySelector("#module-settings-name");
            name.removeAttribute("id");
            name.textContent = modName;
            let settings = modUI.querySelector("#module-settings-content");
            settings.removeAttribute("id");
            let items = data[modName].Config;
            for (var configName in items) {
                settings.appendChild(genConfigUI(configName, items[configName]));
            }
            container.appendChild(modUI);
        }
        let main = document.querySelector("#main");
        let newMain = main.cloneNode(false);
        newMain.appendChild(page);
        main.parentNode.replaceChild(newMain, main);
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
    }).then(function (data) {
        let curLast = document.querySelector("#rp-menu-system-heading");

        function appendMenuEntry(template, selected, prefix, item, itemType) {
            let entry = document.importNode(template.content, true);
            if (selected) {
                entry.querySelector("li").classList.add("pure-menu-selected");
            }
            let link = entry.querySelector("a.pure-menu-link");
            link.href = prefix + item.DirName;
            link.textContent = item.Name;
            let cog = entry.querySelector("a.settings-link");
            cog.addEventListener('click', showSettings);
            cog.dataset.link = "/config" + prefix + item.DirName;
            cog.dataset.name = itemType + " „" + item.Name + " “"

            curLast.parentNode.insertBefore(entry, curLast.nextSibling);
            curLast = curLast.nextSibling;
        }

        let systemTemplate = document.querySelector("#tmpl-system-entry");
        data.Systems.forEach(function (system, index) {
            appendMenuEntry(systemTemplate, index === data.ActiveSystem, "/systems/", system, "System");
        });

        let groupTemplate = document.querySelector("#tmpl-group-entry");
        curLast = document.querySelector("#rp-menu-group-heading");
        data.Groups.forEach(function (group, index) {
            appendMenuEntry(groupTemplate, index === data.ActiveGroup, "/groups/", group, "Group");
        });
        fonts = data.Fonts;
        document.querySelector("#rp-menu-list").style.visibility = "visible";
    }).catch(function (error) {
        console.log(error);
    });
});