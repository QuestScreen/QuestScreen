document.addEventListener("DOMContentLoaded", function () {
    fetch("/static.json").then(function (response) {
        if (response.ok) {
            return response.json();
        }
        throw Error("Could not get static.json");
    }).then(function (data) {
        let curLast = document.querySelector("#rp-menu-system-heading");

        function appendMenuEntry(template, selected, prefix, item) {
            let entry = document.importNode(template.content, true);
            if (selected) {
                entry.querySelector("li").classList.add("pure-menu-selected");
            }
            let link = entry.querySelector("a");
            link.href = prefix + item.DirName;
            link.textContent = item.Name;
            curLast.parentNode.insertBefore(entry, curLast.nextSibling);
            curLast = curLast.nextSibling;
        }

        let systemTemplate = document.querySelector("#tmpl-system-entry");
        data.Systems.forEach(function (system, index) {
            appendMenuEntry(systemTemplate, index === data.ActiveSystem, "/systems/", system);
        });

        let groupTemplate = document.querySelector("#tmpl-group-entry");
        curLast = document.querySelector("#rp-menu-group-heading");
        data.Groups.forEach(function (group, index) {
            appendMenuEntry(groupTemplate, index === data.ActiveGroup, "/groups/", group);
        });
        document.querySelector("#rp-menu-list").style.visibility = "visible";
    }).catch(function (error) {
        console.log(error);
    });
});