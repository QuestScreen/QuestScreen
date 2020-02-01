# QuestScreen

**QuestScreen** is a utility for displaying information during pen & paper
roleplaying sessions. It renders information via SDL, you control it via web
interface. It is designed to be run on boards like the Raspberry Pi.

## Compilation

Dependencies:

 * Go 1.10
   - TODO: modules
 * SDL
   - if you want to run QuestScreen without a window manager, make sure that you
     enable SDL's kmsdrm support (`--enable-video-kmsdrm`). If you want to use
     input with kmsdrm, make sure to link against libudev.

Compile with `make`, install with `make install`.

## Configuration

QuestScreen is configured in `~/.local/share/questscreen`, it looks like this:

    fonts
        <font files>
    base
        config.yaml
        <module configs>
    systems
        <system-name>
            config.yaml
            <module configs>
    groups
        <group-name>
            config.yaml
            <module configs>
    heroes
        <hero-name>
            config.yaml
            <module configs>

`<system-name>`, `<group-name>` and `<hero-name>` each can occur multiple times.
`<module configs>` is a list of data items structured like this:

    <module-name>
        <module specific data>

