# QuestScreen

**QuestScreen** is a utility for displaying information during pen & paper roleplaying sessions.
It renders information via SDL, you control it via web interface.
It is designed to be run on boards like the Raspberry Pi.

## Compilation

[Nix Flakes](https://nixos.wiki/wiki/Flakes) are used for compilation and dependency management.
Without explicitly downloading the source code, you can build QuestScreen via

    nix build github:QuestScreen/QuestScreen

Windows users need to cross-compile from WSL2. TODO: detailed description.

## Documentation

For user documentation, please see the [project's website](https://questscreen.flyx.org/).

## License

This app is licensed under the terms of the [GNU GPL v3](/license-gpl.txt).

Note that the [plugin API](https://github.com/QuestScreen/api) is a separate module licensed under the MIT license so that plugins need not conform to the GNU GPL.