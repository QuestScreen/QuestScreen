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
                // TODO
                let node = document.createElement("P");
                node.textContent = configName + ": " + items[configName].Type;
                settings.appendChild(node);
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
        document.querySelector("#rp-menu-list").style.visibility = "visible";
    }).catch(function (error) {
        console.log(error);
    });
});