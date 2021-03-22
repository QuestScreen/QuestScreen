# Building Quest Screen on the Raspberry Pi

This how-to assumes you use Raspbian.

**Important note for RPi 4 users**:
Quest Screen uses SDL2's OpenGL backend.
You need to enable the experimental OpenGL driver (`raspi-config` -> `Advanced Options` -> `GL driver`).
I suggest enabling the SSH daemon (`Interfacing Options` -> `SSH`) since you might not get a local terminal with the GL driver.
4k@60Hz might not work.
This is all very much experimental.

## SDL 2

First, we need SDL2.
You have two options:

### In a Desktop Environment

If you want to use Quest Screen within a desktop environment, you can use the version supplied with the distribution:

    sudo apt install libsdl2-dev libsdl2-image-dev libsdl2-ttf-dev

### Standalone using KMSDRM

If you want to start Quest Screen without a desktop environment, you need to compile SDL2 yourself because the version in the distribution has this option disabled.
**Important**: Make sure you do not have Raspbian's libsdl2 installed!

First, let's get the dependencies:

```bash
sudo apt install libfreetype6-dev libgl1-mesa-dev libgles2-mesa-dev libdrm-dev libgbm-dev libudev-dev libasound2-dev liblzma-dev libjpeg-dev libtiff-dev libwebp-dev git build-essential
```

Now, compile & install SDL …

```bash
curl -sSL https://libsdl.org/release/SDL2-2.0.12.tar.gz | tar -xz
cd SDL2-2.0.12
./configure --enable-video-kmsdrm --disable-video-rpi --disable-video-x11
make -j$(nproc)
sudo make install
cd ..
```

… SDL_image …

```bash
curl -sSL https://libsdl.org/projects/SDL_image/release/SDL2_image-2.0.5.tar.gz | tar -xz
cd cd SDL2_image-2.0.5
./configure
make -j $(nproc)
sudo make install
cd ..
```

… and SDL_TTF:

```bash
curl -sSL https://libsdl.org/projects/SDL_ttf/release/SDL2_ttf-2.0.15.tar.gz | tar -xz
cd SDL2_ttf-2.0.15
make -j $(nproc)
sudo make install
cd ..
```

## Go

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

## Quest Screen

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