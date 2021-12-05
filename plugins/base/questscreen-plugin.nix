{
	name = "Base";
  description = "QuestScreen's base plugin, providing the standard modules.";
  modules = {
    background.configName = "backgroundConfig";
    herolist = {
      configName = "mConfig";
      config = {
        NameFont = {
          package = "github.com/QuestScreen/api/config";
          type = "FontSelect";
          yamlName = "nameFont";
        };
        DescrFont = {
          package = "github.com/QuestScreen/api/config";
          type = "FontSelect";
          yamlName = "descrFont";
        };
        Background = {
          package = "github.com/QuestScreen/api/config";
          type = "BackgroundSelect";
          yamlName = "background";
        };
      };
    };
    overlays.configName = "overlaysConfig";
    title = {
      configName = "titleConfig";
      config = {
        Font = {
          package = "github.com/QuestScreen/api/config";
          type = "FontSelect";
          yamlName = "font";
        };
        Background = {
          package = "github.com/QuestScreen/api/config";
          type = "BackgroundSelect";
          yamlName = "background";
        };
      };
    };
  };
  templates.groups = [
    {
      name = "Minimal";
      description = "Contains a „Main“ scene with no modules";
      scenes = [{
        name = "Main";
        template = "base.empty";
      }];
    }
    {
      name = "Base";
      description = "Contains a „Main“ scene with base modules";
      scenes = [{
        name = "Main";
        template = "base.default";
      }];
    }
  ];
  templates.scenes = {
    empty = {
      name = "Empty";
      description = "A scene with no modules";
    };
    default = {
      name = "Default";
      description = "A scene with background, title, herolist and overlay";
      config = ''
        modules:
          base.background:
            enabled: true
          base.herolist:
            enabled: true
          base.overlays:
            enabled: true
          base.title:
            enabled: true
      '';
    };
  };
}