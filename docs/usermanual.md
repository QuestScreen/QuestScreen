---
layout: default
title: User Manual
weight: 2
permalink: /usermanual/
---
## Introduction

<aside class="info">

This manual covers the operation of a Quest Screen installation.
For building and installing the app, please see the Readme file on GitHub.

</aside>

Quest Screen is an app that renders information on a screen during pen & paper sessions.
This manual uses the term *session* to refer to a pen & paper session.

Quest Screen is a modular display, meaning that the image rendered on the screen is assembled by a number of independent *modules*.
The set of modules that are currently active is called the current *scene*.
Which modules are available depends on which plugins you have installed.

Whatever the module currently displays is its *state*.
For example, the text displayed by the *title* module is its state.
Moreover, each module has a *configuration* which describes *how* the state is displayed.
For example, the *title* module's configuration contains the font used for displaying its text, and the background color.

During a session, you typically only alter the state of active modules.
Changes to the state will be animated; for example, if you change the text in the *title* module, the old title will move outside of the screen and then the new text will move in.
You can also switch scenes during a session, e.g. for switching between environment view and battle view.
You can freely define any number of scenes and which modules are active in those scenes.

Between sessions, you can modify the configuration of each module.
You can store different configurations per scene, group and/or system.
If you modify the configuration while a scene is displayed, it will be immediately updated without any animation.

You manage your heroes, groups, systems and scenes as *datasets*.
Those can also be created, modified and removed via the web interface.
Quest Screen is not designed to take input from players; only the game master can for example create or remove heroes.

## Running Quest Screen

Quest Screen uses a directory hierarchy to store its configuration and state.
The root directory is located at `~/.local/share/questscreen`.
Before you start it the first time, you need to put some fonts in the subdirectory `fonts` so that it can render text.
Quest Screen does not search the fonts installed on your system so that you won't get a list of epic length when selecting a font.
A nice free font for fantasy-themed groups would be [Garamond](https://garamond.org/).
You can also put symlinks to fonts installed on your system in the directory.

Another important subdirectory is `textures` where you can put grayscale images.
These images will be used to texture backgrounds with two colors, where white is filled with the first color, black with the second, and all gray shades with the corrensponding mixture of the two colors.
The two colors are selectable via the *configuration* interface; you can use a texture with different colors.
The textures should be repeatable in both directions.
Quest Screen does not require any textures to run.

Both fonts and textures are loaded at startup; if you add some, you need to restart Quest Screen.

Quest Screen is designed to be executed on a simple board such as the Raspberry Pi.
Therefore, it takes startup parameters via the command line.
The parameters it takes are:

 * `-f`, `--fullscreen`: Run in full screen. You typically want to use that flag.
   Running Quest Screen windowed is only useful for debugging.
 * `-p <num>`, `--port <num>`: Instruct Quest Screen to run the web interface on port `<num>`.
   Default is `8080`.

After starting Quest Screen, you can control it via the web interface.
Any keyboard input on the host system will cause a popup to appear asking you whether you want to quit (this won't work if no fonts are installed because then it can't render any text).
The startup screen will tell you the IP address and port under which the web interface is available.
The host obviously must be connected to a network; whether the displayed URL works depends on your network setup, but it should suffice for a simple home network.

## Managing Datasets

The web interface allows you to manage your *datasets*, i.e. systems, groups, scenes and heroes.
Scenes and heroes always belong to a group, a group may link to a system.
Sometimes plugins require a system to exist, in which case you cannot remove it as long as the plugin is installed.

When you create a new group or scene, you can select between a number of templates.
Templates are defined by plugins and are used to provide default configurations which you can then modify.
For example, a plugin might provide a group template which contains a default scene (with background, list of heroes and so on) and a battle scene which contains the module showing battle stats that is provided by the plugin.

The scene templates contained in a group template are also available for creating single scenes.
The *base* plugin (which provides basic modules that are always available) provides empty templates if you want to start with an empty group or scene.

## Configuring the Modules

By default, modules have a pretty simple look.
For example, the *title* module by default has a white background.
You can change the module's look in the *configuration* section.

Configuration has multiple layers.
On each layer, you may define a configuration that overrides the one from a lower layer.
If you don't set a configuration for a layer, the configuration from the lower layer will be used.
The following layers exist:

 * The default configuration is defined for each module and is not editable.
 * The base configuration applies to all groups and overrides the default configuration.
 * The system configuration applies to all groups linked to that system and overrides the base configuration.
 * The group configuration applies to all scenes in that group and overrides the system configuration (or the base configuration if the group is not linked to a system).
 * The scene configuration applies to a single scene and overrides the group configuration.

Usually, you want to edit the base or system configuration unless you want to have a different look for each group.
Scene configuration is useful if you want to use a module in multiple scenes but want to have it look different.

## The State: Using Quest Screen during a session

Finally, when you have created a group and configured it to your liking, you can start using Quest Screen during a session.
You start by selecting the active group.
Only one group can be active and you can only modify the state of the active group.

Unlike the configuration and dataset pages (where you can click *reset* or *save* to commit your changes), changes to the state  are typically sent immediately.
Each action triggers the corresponding animation.
Which interactions are available depends on the active modules in the current scene.
Changing the scene will modify the possible interactions depending on the modules in the new scene.
Module states are local to the scene, so if you have the *title* module active in two scenes, updating the text in one scene will not modify the text displayed in the other scene.

## Providing files to modules

Some modules, like the *background* module, depend on files (in this case, images).
Currently, the web interface does not allow file uploads; you must place them on the host system manually.
The files for each module have to be put in a directory whose name matches the module's ID.
You can look up the ID of a module on the *datasets* page in the web interface.

You can create such a directory in any of the following places (relative to the root directory `~/.local/share/questscreen`):

 * `base`: Files placed will be available in all groups.
 * `systems/<system-id>`: Files placed here will be available in all groups that are linked to the system identified by `<system-id>`.
 * `groups/<group-id>`: Files placed here will be available in all scenes of the group identified by `<group-id>`.
 * `groups/<group-id>/scenes/<scene-id>`: Files placed here will be available in the scene identified by `<scene-id>`.

So for example, if you have the following files:

    base/background/one.jpg
    groups/firstgroup/background/two.jpg
    groups/othergroup/scenes/primary/background/three.jpg

The background module in any scene of `firstgroup` will have access to `one.jpg` and `two.jpg`, while in the `primary` scene if group `othergroup`, it will have access to `one.jpg` and `three.jpg`.

Modules may define multiple sets of files that need to be placed in separate subdirectories with defined names.
They may also require a specific file with a defined name.
Make sure to read the documentation of the plugin providing the module to figure out what files it expects and where you have to put them!

## Persistence

Whenever you modify the state, the active scene of a group, any configuration or any dataset, the change will be immediately written to the file system.
When you quit Quest Screen and later start it again, any group will still be in the state where you left it.

Configuration will be written to `config.yaml` files in the corresponding base, system, group or scene directory.
State will be written to `state.yaml` in the scene directory.

If you want to backup your state, you can simply backup the whole base directory.
This will save your whole configuration and state, including all additional files you provide to modules.

## Manage Plugins

Plugins are placed in the `plugins` directory inside the root directory.
They are loaded at startup; you can't add a plugin while Quest Screen is running.

Plugins provide you with additional modules, additional systems and/or additional group and scene templates.
Writing plugins is covered by the [plugin tutorial](/plugins/).