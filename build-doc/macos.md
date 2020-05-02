# Building Quest Screen on macOS

You will need the SDL2 library and the Go compiler.
Assuming you have [Homebrew](https://brew.sh/) installed, just do

```bash
brew install sdl2 sdl2_image sd2_ttf go
```

Then fetch a build-time dependency and the main sources:

```bash
go get github.com/go-bindata/go-bindata/...
go get -d github.com/QuestScreen/QuestScreen
```

You will get a `can't load package:` warning here because Go's packaging system is badly designed.
Ignore it.
To build Quest Screen, do:

```bash
cd ~/go/src/github.com/QuestScreen/QuestScreen
make
```

You can then launch Quest Screen via `./questscreen`.
Currently, there's no support for building an app bundle.