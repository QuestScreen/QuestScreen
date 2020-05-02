# QuestScreen

**QuestScreen** is a utility for displaying information during pen & paper roleplaying sessions.
It renders information via SDL, you control it via web interface.
It is designed to be run on boards like the Raspberry Pi.

## Compilation

Dependencies that need to be installed manually (`go.mod` takes care of required Go modules):

 * **Go 1.12** or later
 * **SDL2**, **SDL2_image**, **SDL2_ttf**

   If you want to run QuestScreen without a window manager, make sure that you enable SDL's kmsdrm support (`--enable-video-kmsdrm`).
   If you want to use input with kmsdrm, make sure to link against libudev.

 * **go-bindata**

   Install with `go get github.com/go-bindata/go-bindata/...`.
   This is used for including web-related files (html, css, js) in the binary.
   Since this is a compile-time only dependency, it is not listed in `go.mod`.

 * **git**

   Used to autogenerate the current version string when building.
   This requires any build to happen from within a git repository.

Compile with `make`, install with `make install`.

### Building on Raspberry Pi

**Important note for RPi 4 users**:
Quest Screen uses SDL2's OpenGL backend.
You need to enable the experimental OpenGL driver (`raspi-config` -> `Advanced Options` -> `GL driver`).
I suggest enabling the SSH daemon (`Interfacing Options` -> `SSH`) since you might not get a local terminal with the GL driver.
4k@60Hz might not work.
I am still experimenting with it.

#### SDL 2

First, we need SDL2.
You have two options:

 * If you want to use Quest Screen within a desktop environment, you can use the version supplied with the distribution:

    sudo apt install libsdl2-dev libsdl2-image-dev libsdl2-ttf-dev

 * If you want to start Quest Screen without a desktop environment, you need to compile it yourself because the version in the distribution has this option disabled.
   **Important**: Make sure you do not have Raspbian's libsdl2 installed!

   First, let's get the dependencies:

    sudo apt install libfreetype6-dev libgl1-mesa-dev libgles2-mesa-dev libdrm-dev libgbm-dev libudev-dev libasound2-dev liblzma-dev libjpeg-dev libtiff-dev libwebp-dev git build-essential

   Now, compile & install SDL …

    curl -sSL https://libsdl.org/release/SDL2-2.0.12.tar.gz | tar -xz
    cd SDL2-2.0.12
    ./configure --enable-video-kmsdrm --disable-video-rpi --disable-video-x11
    make -j$(nproc)
    sudo make install
    cd ..

  … SDL_image …

    curl -sSL https://libsdl.org/projects/SDL_image/release/SDL2_image-2.0.5.tar.gz | tar -xz
    cd cd SDL2_image-2.0.5
    ./configure
    make -j $(nproc)
    sudo make install
    cd ..

  … and SDL_TTF:

    curl -sSL https://libsdl.org/projects/SDL_ttf/release/SDL2_ttf-2.0.15.tar.gz | tar -xz
    cd SDL2_ttf-2.0.15
    make -j $(nproc)
    sudo make install
    cd ..

#### Go

Since Raspbian buster only provides an ancient Go version, we'll need to fetch our Go compiler from Google (alternatively you can compile it from source if you want).
**Important**: If you have the `golang` package installed, you must remove it with `sudo apt remove golang && sudo apt autoremove`.

```bash
sudo rm -rf /usr/local/go
curl -sSL https://dl.google.com/go/go1.14.2.linux-armv6l.tar.gz | sudo tar -xz -C /usr/local
cat << EOF > .goenv
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
EOF
source .goenv
```

If you want to do more things with go, you can put the two `export` lines into your `.bashrc` instead.

#### Quest Screen

Now, let's fetch one last build-time dependency and then the Quest Screen sources themselves:

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

This should give you a `questscreen` executable.
RPi users might need to run the executable once to get a default `config.yaml` and then tinker with the size settings / fullscreen flag.
See the [user manual](https://questscreen.flyx.org/usermanual/) for details.

## Documentation

For user documentation, please see the [project's website](https://questscreen.flyx.org/).

## License

This app is licensed under the terms of the [GNU GPL v3](/license-gpl.txt).

Note that the [plugin API](https://github.com/QuestScreen/api) is a separate module licensed under the MIT license so that plugins need not conform to the GNU GPL.