{
  description = "A kiosk application for use during pen & paper sessions, controlled via web interface";
  inputs = {
    nixpkgs.url = github:NixOS/nixpkgs/nixos-21.11;
    flake-utils.url = github:numtide/flake-utils;
  };
  outputs = { self, nixpkgs, flake-utils }: flake-utils.lib.eachSystem flake-utils.lib.allSystems (system:
    let
      native = import nixpkgs { inherit system; };
      buildGoModule = native.buildGo117Module;
      
      gopherJS = buildGoModule {
        src = native.fetchFromGitHub {
          owner = "gopherjs";
          repo = "gopherjs";
        };
      };
      goimports = buildGoModule rec {
        pname = "goimports";
        version = "v0.1.8";
        rev = "e212aff8fd146c44ddb0167c1dfbd5531d6c9213";
        subPackages = [ "cmd/goimports" ];
        vendorSha256 = "7YocW8o4J2JZqb1uZgCQmfaQJN1lsrteDZKLyPk2/f8=";
        src = native.fetchgit {
          inherit rev;
          url = "https://go.googlesource.com/tools";
          sha256 = "sha256-548eisukLCoLY5LuESUfgCzqiVbPXy9J+sgHn/W5MUE=";
        };
      };
      
      basePlugin = import "${self}/plugins/base/metadata.nix" self;
      
      loadPlugin = {index, plugin, id, configTypes}:
        let
          scenes = plugin.templates.scenes or {};
        in with builtins; {
          inherit (plugin) name description cssFiles goImportPath;
          inherit id;
          source = plugin.outPath;
          modules = native.lib.imap1 (mIndex: {name, value}: {
            inherit name;
            inherit (value) configName;
            config = mapAttrs (k: v: v // {
              packageImportName = (getAttr v.package configTypes);
            }) (value.config or {});
            importName = "p${toString index}m${toString mIndex}";
          }) (native.lib.mapAttrsToList (k: v: {name = k; value = v;}) plugin.modules);
          templates = {
            systems = plugin.systems or [];
            groups = plugin.groups or [];
            scenes = map (key: (getAttr key scenes) // {id = key;}) (attrNames scenes);
          };
        };
      injectWeb = path: if native.lib.hasSuffix "/config" path then
        (native.lib.removeSuffix "/config" path) + "/web/config" else
        throw "invalid config path (does not end with '/config'): ${path}";
      renderImports = prefix: plugins: let
        importLine = plugin: module: ''${module.importName} "${plugin.goImportPath}/${prefix}${module.name}"'';
        importLines = plugin: (builtins.foldl' (a: b: a + "\n\t" + (importLine plugin b)) "" plugin.modules);
      in builtins.foldl' (a: b: a + (importLines b)) "" plugins;
      pluginCode = (import plugins/plugins.go.nix) (native.lib // { inherit renderImports; });
      webPluginCode = (import web/main/plugins.go.nix) (native.lib // { inherit renderImports; });
      webConfigCode = (import web/configDescr.go.nix) (native.lib // { inherit injectWeb; });
      
      vendoredSources = {plugins, configTypes}: native.stdenvNoCC.mkDerivation {
        name = "questscreen-vendored-sources";
        src = builtins.filterSource (path: type: !(builtins.foldl' (x: y: x || native.lib.hasSuffix y path) false [ ".nix" ".md" ".lock" ])) self;
        buildInputs = [ native.go ];
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
          ${native.vend}/bin/vend
        '';
        installPhase = ''
          ln -s . src
          tar czf $out --exclude=src/src src/*
        '';
      };
      
      askew = buildGoModule {
        pname = "askew";
        version = "0.1.0";
        propagatedBuildInputs = [ goimports ];
        src = native.fetchFromGitHub {
          owner = "flyx";
          repo = "askew";
          rev = "6d94175d9696a7c13d7e331411a633848c8338d4";
          sha256 = "3OOmyiqT8GUQFfQn7veHUiyGA0MHB+xewcqN5ymKpls=";
        };
        subPackages = [ "." ];
        vendorSha256 = "oQiZNhbjCpLBPSuzOssGYJoMEe0i7xVeqc3O1LJxMy0=";
      };
      asiteCode = (import web/site/main.asite.nix) native.lib;
  
      questscreen-webui = {plugins, wasm, sources}: buildGoModule {
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
          ${askew}/bin/askew -o assets -b ${if wasm then "wasm" else "gopherjs"} \
            --exclude app,assets,build-doc,data,display,main,shared,vendor .
          printenv PLUGIN_CODE > plugins/plugins.go
        '';
        buildPhase = if wasm then ''
          (cd web/main && env GOOS=js GOARCH=wasm ${native.go}/bin/go build -o main.wasm)
        '' else ''
          (cd web/main && env GOOS=linux ${gopherJS}/bin/gopherjs build)
        '';
        doCheck = false;
        installPhase = if wasm then ''
          mkdir -p $out/web/assets
          cp -t $out/web/assets web/main/main.wasm assets/index.html \
            "$(${native.go}/bin/go env GOROOT)/misc/wasm/wasm_exec.js"
        '' else ''
          mkdir -p $out/web/assets
          cp -t $out/web/assets web/main/main.js* assets/index.html
        '';
      };
      questscreen-builder = {pkgs, wasm, exeName ? "questscreen", plugins}:
        let
          # enumerates all used config types
          configTypes = with builtins; let
            moduleCfgPackages = module: native.lib.mapAttrsToList (k: v: v.package) (module.config or {});
            pluginCfgPackages = plugin: let modules = plugin.modules or {}; in foldl'
              (cur: new: cur ++ (moduleCfgPackages (getAttr new modules)))
              [] (attrNames modules);
            cfgPackages = foldl' (cur: new: cur ++ (pluginCfgPackages plugins.${new})) []
              (attrNames plugins);
            value = foldl'
              (cur: new: if hasAttr new cur then cur else
                cur // {"${new}" = "cfg${toString ((length (attrNames cur)) + 1)}";})
              {} cfgPackages;
          in value;
          loadedPlugins = native.lib.imap1
            (index: id: loadPlugin {inherit index id configTypes; plugin = plugins.${id};})
            (builtins.attrNames plugins);
          sources = vendoredSources {inherit configTypes; plugins = loadedPlugins;};
          compiledWebUI = questscreen-webui {inherit sources wasm; plugins = loadedPlugins;};
          suffix = if pkgs.stdenv.hostPlatform.isWindows then ".exe" else "";
          pluginAssets = plugin: ''cp -r -T ${plugin.source}/web/assets assets/${plugin.id}'';
        in pkgs.buildGo117Module {
          pname = exeName;
          version = self.shortRev or "dirty-${self.lastModifiedDate}";
          src = sources;
          modRoot = "src";
          subPackages = [ "main" ];
          vendorSha256 = null;
          nativeBuildInputs = [ pkgs.pkg-config ];
          buildInputs = [ compiledWebUI pkgs.SDL2 pkgs.SDL2_ttf pkgs.SDL2_image ];
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
      addWithPlugins = f: origArgs: let
        origRes = f origArgs;
      in 
        origRes // {
          withPlugins = pluginList: addWithPlugins
            (f (origArgs // {plugins = origArgs.plugins // pluginList;}));
        };
    in rec {
      packages = {
        questscreen = addWithPlugins questscreen-builder {
          pkgs = native; wasm = true; plugins = {base = basePlugin;};
        };
        questscreen-js = addWithPlugins questscreen-builder {
          pkgs = native; wasm = false; plugins = {base = basePlugin;};
        };
      };
      defaultPackage = packages.questscreen;
    });
}