# QuestScreen

**QuestScreen** is a utility for displaying information during pen & paper roleplaying sessions.
It renders information via SDL, you control it via web interface.
It is designed to be run on boards like the Raspberry Pi.

## Compilation

Build dependencies:

 * **Go 1.10**

   TODO: modules
 * **SDL**

   If you want to run QuestScreen without a window manager, make sure that you enable SDL's kmsdrm support (`--enable-video-kmsdrm`).
   If you want to use input with kmsdrm, make sure to link against libudev.

 * **go-bindata**

   Install with `go get github.com/go-bindata/go-bindata/...`.
   This is used for including web-related files (html, css, js) in the binary.

 * **git**

   Used to autogenerate the current version string when building.
   This requires any build to happen from within a git repository.

Compile with `make`, install with `make install`.

## Documentation

For user documentation, please see the [project's website](questscreen.github.io/questscreen).

## License

The `api` package that can be used to develop plugins for QuestScreen, as well as all files in the `web` directory, are licensed under terms of the [MIT license](/license-mit.txt).
This ensures that you can distribute QuestScreen plugins under any license.

All other packages, which constitute the main application, are licensed under the terms of the [GNU GPL v3](/license-gpl.txt).
