{
  description = "A kiosk application for use during pen & paper sessions, controlled via web interface";
  inputs = {
    nixpkgs.url    = github:NixOS/nixpkgs/nixos-21.11;
    utils.url      = github:numtide/flake-utils;
    nix-filter.url = github:numtide/nix-filter;
    #api.url        = github:QuestScreen/api;
    zigf.url       = github:arqv/zig-overlay;
    zigf.inputs.nixpkgs.follows = "nixpkgs";
    go-1-18.url    = github:flyx/go-1.18-nix;
  };
  outputs = { self, nixpkgs, utils, nix-filter, zigf, go-1-18 }: let
    go118pkgs = system: import nixpkgs {
      inherit system;
      overlays = [ go-1-18.overlay ];
    };
    platforms = system: let
      pkgs = go118pkgs system;
      zig = zigf.packages.${system}."0.9.1".overrideAttrs (old: {
        installPhase = let
          armFeatures = builtins.fetchurl {
            url = "https://sourceware.org/git/?p=glibc.git;"    +
                  "a=blob_plain;f=sysdeps/arm/arm-features.h;"  +
                  "h=80a1e2272b5b4ee0976a410317341b5ee601b794;" +
                  "hb=0281c7a7ec8f3f46d8e6f5f3d7fca548946dbfce";
            name = "glibc-2.35_arm-features.h";
            sha256 =
              "1g4yb51srrfbd4289yj0vrpzzp2rlxllxgz8q4a5zw1n654wzs5a";
          };
        in old.installPhase + "\n cp ${armFeatures} $out/lib/libc/glibc/sysdeps/arm/arm-features.h";
      });
      fromDebs = name: debSources: pkgs.stdenvNoCC.mkDerivation rec {
        inherit name;
        srcs = with builtins; map fetchurl debSources;
        phases = [ "unpackPhase" "installPhase" ];
        nativeBuildInputs = [ pkgs.dpkg ];
        unpackPhase = builtins.concatStringsSep "\n" (builtins.map
          (src: "${pkgs.dpkg}/bin/dpkg-deb -x ${src} .") srcs);
        installPhase = ''
          mkdir -p $out
          cp -r * $out
        '';
      };
      fromPacman = name: pmSources: pkgs.stdenvNoCC.mkDerivation rec {
        inherit name;
        srcs = with builtins; map fetchurl pmSources;
        phases = [ "unpackPhase" "installPhase" ];
        nativeBuildInputs = [ pkgs.gnutar pkgs.zstd ];
        unpackPhase = builtins.concatStringsSep "\n" (builtins.map
          (src: ''
            ${pkgs.gnutar}/bin/tar -xvpf ${src} --exclude .PKGINFO \
              --exclude .INSTALL --exclude .MTREE --exclude .BUILDINFO
          '') srcs);
        installPhase = ''
          mkdir -p $out
          cp -r -t $out *
        '';
      };
      platformConfig = {
        zigTarget, deps, CGO_CPPFLAGS, CGO_LDFLAGS, GOOS, GOARCH, overrides ? _: {}
      }: {
        inherit deps;
        buildGoModuleOverrides = old: let
          basic = {
            CGO_ENABLED = true;
            inherit CGO_CPPFLAGS CGO_LDFLAGS;
            preBuild = ''
              export ZIG_LOCAL_CACHE_DIR=$(pwd)/zig-cache
              export ZIG_GLOBAL_CACHE_DIR=$ZIG_LOCAL_CACHE_DIR
              export CC="${zig}/bin/zig cc -target ${zigTarget}"
              export CXX="${zig}/bin/zig c++ -target ${zigTarget}"
              export GOOS=${GOOS}
              export GOARCH=${GOARCH}
              export CGO_CPPFLAGS=${CGO_CPPFLAGS}
              export CGO_LDFLAGS=${CGO_LDFLAGS}
            '';
            overrideModAttrs = _: {};
          };
        in basic // (overrides (old // basic));
      };
      msysPrefix = "https://mirror.msys2.org/mingw/clang64/" +
                   "mingw-w64-clang-x86_64-";
      rpiPrefix = "http://ftp.de.debian.org/debian/pool/main/libs/";
      includeFlags = pkgsToInclude: builtins.concatStringsSep " "
        (builtins.map (pkg: "-I${pkg}/usr/include") pkgsToInclude);
      ldFlags = pkgsToLink: builtins.concatStringsSep " "
        (builtins.map (pkg: "-L${pkg}/usr/lib/arm-linux-gnueabihf -l${pkg.name}")
        pkgsToLink);
    in {
      raspberryPi4 = platformConfig rec {
        zigTarget = "arm-linux-gnueabihf";
        deps = [
          (fromDebs "SDL2" [{
            url = "${rpiPrefix}/libsdl2/libsdl2-2.0-0_2.0.14+dfsg2-3_armhf.deb";
            sha256 = nixpkgs.lib.fakeSha256;
          } {
            url = "${rpiPrefix}/libsdl2/libsdl2-dev_2.0.14+dfsg2-3_armhf.deb";
            sha256 = nixpkgs.lib.fakeSha256;
          }])
          (fromDebs "SDL2_ttf" [{
            url = "${rpiPrefix}/libsdl2-ttf/libsdl2-ttf-2.0-0_2.0.15+dfsg1-1_armhf.deb";
            sha256 = nixpkgs.lib.fakeSha256;
          } {
            url = "${rpiPrefix}/libsdl2-ttf/libsdl2-ttf-dev_2.0.15+dfsg1-1_armhf.deb";
            sha256 = nixpkgs.lib.fakeSha256;
          }])
          (fromDebs "SDL2_image" [{
            url = "${rpiPrefix}/libsdl2-image/libsdl2-image-2.0-0_2.0.5+dfsg1-2_armhf.deb";
            sha256 = nixpkgs.lib.fakeSha256;
          } {
            url = "${rpiPrefix}/libsdl2-image/libsdl2-image-dev_2.0.5+dfsg1-2_armhf.deb";
            sha256 = nixpkgs.lib.fakeSha256;
          }])
        ];
        CGO_CPPFLAGS = includeFlags deps;
        CGO_LDFLAGS  = ldFlags deps;
        GOOS = "linux";
        GOARCH = "arm";
      };
      win64 = platformConfig rec {
        zigTarget = "x86_64-windows-gnu";
        deps = let
          depLines = with nixpkgs.lib; splitString "\n" (fileContents ./win64-deps.txt);
        in with builtins; map (
          line: let
            parts = elemAt (nixpkgs.lib.splitString " " line);
            name = elemAt (split "(.*?)-[0-9].*" (parts 1)) 1;
          in fromPacman name [{
            url = "${msysPrefix}${parts 1}";
            sha256 = parts 0;
          }]
        ) depLines;
        CGO_CPPFLAGS = includeFlags deps;
        CGO_LDFLAGS  = ldFlags deps;
        GOOS = "windows";
        GOARCH = "amd64";
        overrides = _: with builtins; {
          postInstall = ''
            cp -t $out/bin/windows_amd64 {${concatStringsSep "," (attrValues deps)}}/clang64/bin/*.dll
          '';
        };
      };
    };
    buildQuestScreen = {
      system, pname, version, wasm ? true, exeName ? "questscreen", plugins ? {},
      deps ? { inherit (go118pkgs system) SDL2 SDL2_ttf SDL2_image; },
      buildGoModuleOverrides ? _: {}
    }: let
      pkgs = go118pkgs system;
      loadPlugin = {index, plugin, id, configTypes}:
        let
          scenes = plugin.templates.scenes or {};
          systems = plugin.templates.systems or {};
        in with builtins; {
          inherit (plugin) name version label description cssFiles goImportPath;
          inherit id;
          source = plugin.outPath;
          modules = pkgs.lib.imap1 (mIndex: {name, value}: {
            inherit name;
            inherit (value) configName;
            config = mapAttrs (k: v: v // {
              packageImportName = (getAttr v.package configTypes);
            }) (value.config or {});
            importName = "p${toString index}m${toString mIndex}";
          }) (pkgs.lib.mapAttrsToList (k: v: {name = k; value = v;}) plugin.modules);
          templates = {
            systems = map (key: (getAttr key systems) // {id = key;}) (attrNames systems);
            groups = plugin.templates.groups or [];
            scenes = map (key: (getAttr key scenes) // {id = key;}) (attrNames scenes);
          };
        };
      injectWeb = path: if pkgs.lib.hasSuffix "/config" path then
        (pkgs.lib.removeSuffix "/config" path) + "/web/config" else
        throw "invalid config path (does not end with '/config'): ${path}";
      renderImports = prefix: plugins: let
        importLine = plugin: module: ''${module.importName} "${plugin.goImportPath}/${prefix}${module.name}"'';
        importLines = plugin: (builtins.foldl' (a: b: a + "\n\t" + (importLine plugin b)) "" plugin.modules);
      in builtins.foldl' (a: b: a + (importLines b)) "" plugins;
      pluginCode = (import plugins/plugins.go.nix) (pkgs.lib // { inherit renderImports; });
      webPluginCode = (import web/main/plugins.go.nix) (pkgs.lib // { inherit renderImports; });
      webConfigCode = (import web/configDescr.go.nix) (pkgs.lib // { inherit injectWeb; });
      
      gopherJS = pkgs.buildGo118Module {
        pname = "gopher-js";
        version = "1.17.1+go1.17.3";
        src = pkgs.fetchFromGitHub {
          owner = "gopherjs";
          repo = "gopherjs";
          rev = "ed9a9b14a74738df4185b7627b276902ad07d06f";
          sha256 = "sha256-YZFYqTQaLt4B0Hu/UznfGvQVjd0UaVlqjd1D+514xu0=";
        };
        vendorSha256 = "sha256-gio7tA0VrzPOoDkIW5iFr65NFuDLMpbf4pR9rdU8p8Y=";
        checkPhase = "true";
      };
      goimports = pkgs.buildGo118Module rec {
        pname = "goimports";
        version = "v0.1.8";
        rev = "e212aff8fd146c44ddb0167c1dfbd5531d6c9213";
        subPackages = [ "cmd/goimports" ];
        vendorSha256 = "UbQrjMv5x2zREeHDlIffri6PylE75vEwD9n0S3XyC8I=";
        src = pkgs.fetchgit {
          inherit rev;
          url = "https://go.googlesource.com/tools";
          sha256 = "sha256-548eisukLCoLY5LuESUfgCzqiVbPXy9J+sgHn/W5MUE=";
        };
      };
      
      vendoredSources = {plugins, configTypes}: let
        drv = pkgs.stdenvNoCC.mkDerivation {
          name = "questscreen-vendored-sources";
          src = nix-filter.lib.filter {
            root = ./.;
            exclude = [ ./flake.nix ./flake.lock ./Readme.md ./license.md ];
          };
          buildInputs = [ pkgs.go_1_18 ];
          phases = [ "unpackPhase" "configurePhase" "buildPhase" "installPhase" ];
          VERSIONINFO_CODE = ''
            package versioninfo
            
            var CurrentVersion = "${self.shortRev or "dirty-${self.lastModifiedDate}"}"
            var Date = "${self.lastModifiedDate}"
          '';
          PLUGIN_CODE = pluginCode plugins;
          WEB_PLUGIN_CODE = webPluginCode plugins;
          WEB_CONFIG_CODE = webConfigCode plugins configTypes;
          configurePhase = ''
            mkdir -p versioninfo
            printenv VERSIONINFO_CODE > versioninfo/versioninfo.go
            printenv PLUGIN_CODE > plugins/plugins.go
            printenv WEB_PLUGIN_CODE > web/main/plugins.go
            printenv WEB_CONFIG_CODE > web/configDescr.go
          '';
          buildPhase = ''
            export GOCACHE=$TMPDIR/go-cache
            export GOPATH="$TMPDIR/go"
            ${pkgs.vend}/bin/vend
          '';
          installPhase = ''
            ln -s . src
            tar czf $out --exclude=src/src src/*
          '';
        };
      in builtins.trace "generated sources derivation: ${drv}" drv;
      
      askew = pkgs.buildGo117Module {
        pname = "askew";
        version = "0.1.0";
        propagatedBuildInputs = [ goimports ];
        src = pkgs.fetchFromGitHub {
          owner = "flyx";
          repo = "askew";
          rev = "3986345cbbd3c5e52f91a1d6a08b62ed14088b45";
          sha256 = "sha256-3NIghslhcLSgOIqPjbmfdpZvyJNiDgCTqqT+lGYcFGY=";
        };
        subPackages = [ "." ];
        vendorSha256 = "oQiZNhbjCpLBPSuzOssGYJoMEe0i7xVeqc3O1LJxMy0=";
      };
      asiteCode = (import web/site/main.asite.nix) pkgs.lib;
      
      questscreen-webui = {plugins, wasm, sources}: pkgs.buildGo117Module {
        pname = "questscreen-webui";
        version = self.shortRev or "dirty-${self.lastModifiedDate}";
        src = sources;
        modRoot = "src";
        subPackages = [ "web/main" ];
        overrideModAttrs = old: {
          unpackPhase = "tar zxf $src && echo unpacked: $src";
        };
        vendorSha256 = null;
        buildInputs = [ askew ] ++ (if wasm then [] else [ gopherJS ]);
        unpackPhase = "tar zxf $src";
        ASITE_CODE = asiteCode plugins;
        PLUGIN_CODE = pluginCode plugins;
        postConfigure = ''
          printenv ASITE_CODE > web/site/main.asite
          mkdir -p $GOPATH/bin
          ln -s ${goimports}/bin/goimports $GOPATH/bin/goimports
          ln -s ${pkgs.go_1_18}/bin/go $GOPATH/bin/go
          ${askew}/bin/askew -o assets -b ${if wasm then "wasm" else "gopherjs"} \
            --exclude app,assets,build-doc,data,display,main,shared,vendor .
          printenv PLUGIN_CODE > plugins/plugins.go
        '';
        buildPhase = if wasm then ''
          export GOCACHE=$TMPDIR/go-cache
          (cd web/main && env GOOS=js GOARCH=wasm ${pkgs.go_1_18}/bin/go build -o main.wasm)
        '' else ''
          mkdir -p home # gopherjs does not honor GOCACHE for some reason.
                        # therefore we redirect writes to the gopherjs cache here.
          (export HOME=$(pwd)/home && cd web/main && env \
            GOOS=linux \
            GOPHERJS_GOROOT="$(${pkgs.go_1_18}/bin/go env GOROOT)" \
            ${gopherJS}/bin/gopherjs build)
        '';
        doCheck = false;
        installPhase = if wasm then ''
          mkdir -p $out/web/assets
          cp -t $out/web/assets web/main/main.wasm assets/index.html \
            "$(${pkgs.go_1_17}/bin/go env GOROOT)/misc/wasm/wasm_exec.js"
        '' else ''
          mkdir -p $out/web/assets
          cp -t $out/web/assets web/main/main.js* assets/index.html
        '';
      };
      
      pluginsWithBase = plugins // { base = import "${self}/plugins/base/metadata.nix" self; };
      configTypes = with builtins; let
        moduleCfgPackages = module: pkgs.lib.mapAttrsToList (k: v: v.package) (module.config or {});
        pluginCfgPackages = plugin: let modules = plugin.modules or {}; in foldl'
          (cur: new: cur ++ (moduleCfgPackages (getAttr new modules)))
          [] (attrNames modules);
        cfgPackages = foldl' (cur: new: cur ++ (pluginCfgPackages pluginsWithBase.${new})) []
          (attrNames pluginsWithBase);
        value = foldl'
          (cur: new: if hasAttr new cur then cur else
            cur // {"${new}" = "cfg${toString ((length (attrNames cur)) + 1)}";})
          {} cfgPackages;
      in value;
      loadedPlugins = pkgs.lib.imap1
        (index: id: loadPlugin {inherit index id configTypes; plugin = pluginsWithBase.${id};})
        (builtins.attrNames pluginsWithBase);
      sources = vendoredSources {inherit configTypes; plugins = loadedPlugins;};
      compiledWebUI = let
        ui = questscreen-webui {inherit sources wasm; plugins = loadedPlugins;};
      in builtins.trace "WebUI derivation: ${ui}" ui;
      suffix = if pkgs.stdenv.hostPlatform.isWindows then ".exe" else "";
      pluginAssets = plugin: ''cp -r -T ${plugin.source}/web/assets assets/${plugin.id}'';
    in pkgs.buildGo118Module {
      inherit pname version;
      src = sources;
      modRoot = "src";
      subPackages = [ "main" ];
      vendorSha256 = null;
      nativeBuildInputs = [ pkgs.pkg-config ];
      buildInputs = [ compiledWebUI ] ++ (builtins.attrValues deps);
      overrideModAttrs = old: {
        unpackPhase = "tar xzf $src";
      };
      unpackPhase = "tar xzf $src";
      postConfigure = ''
        cp -t assets ${compiledWebUI}/web/assets/* vendor/github.com/QuestScreen/api/web/assets/* \
          web/assets/*
        ${builtins.concatStringsSep "\n" (map pluginAssets loadedPlugins)}
      '';
      preBuild = ''
        export CGO_CFLAGS=$(pkg-config --cflags sdl2 sdl2_image sdl2_ttf)
      '';
      postBuild = ''
        mv "$GOPATH/bin/main${suffix}" "$GOPATH/bin/questscreen${suffix}"
      '';
      postInstall = ''
        mkdir -p $out/share
        cp -r -t $out/share resources/*
      '';
    };
  in with utils.lib; eachSystem allSystems (system: let
    pkgs = import nixpkgs { inherit system; };
  in rec {
    packages = {
      questscreen = buildQuestScreen {
       inherit system;
        pname = "questscreen";
        version = self.shortRev or "dirty-${self.lastModifiedDate}";
      };
      questscreen-js = buildQuestScreen {
        inherit system;
        pname = "questscreen-js";
        version = self.shortRev or "dirty-${self.lastModifiedDate}";
        wasm = false;
      };
    };
    defaultPackage = packages.questscreen;
  }) // {
    lib.buildQuestScreen = buildQuestScreen;
  };
}