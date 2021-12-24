{imap0, renderImports, ...}: plugins: with builtins; ''package plugins

// Code generated by nix. DO NOT EDIT.

import (
	"github.com/QuestScreen/api/modules"
	"github.com/QuestScreen/QuestScreen/app"
	${renderImports "" plugins}
)

func LoadPlugins(a app.App) {
	${
		let
			nonEmpty = msg: a: if a == "" then throw msg else a;
			parseTmplRef = ref: msg: i: (
				(splitted: nonEmpty msg (elemAt splitted i))
				((res: if (length res) == 3 && (elemAt res 0) == "" && (elemAt res 2) == "" then
						(elemAt res 1)
					else throw msg) (split "([[:alnum:]]+)\.([[:alnum:]]+)" ref)
				)
			);
			findWithIndex = msg: list: id: let
				res = filter (a: a.item.id == id) (imap0 (index: b: {inherit index; item = b;}) list);
			in if (length res) == 0 then throw msg else elemAt res 0;
			
			resolveSceneTmplRef = ref: let
				parsed = parseTmplRef ref "invalid template reference: \"${ref}\"";
				refPlugin = findWithIndex "unknown template ID: ${parsed 0}" plugins (parsed 0);
				refTemplate = findWithIndex "unknown module ID in template ${parsed 0}: ${parsed 1}" (refPlugin.item.templates.scenes or []) (parsed 1);
			in {
				pluginIndex = refPlugin.index;
				templateIndex = refTemplate.index;
			};
			systemTemplate = system: ''{
				ID: ${toJSON system.id},
				Name: ${toJSON system.name},
				Config: []byte(`${system.config or ""}`),
			},'';
			sceneTemplateRefs = scene:
				let
					ref = resolveSceneTmplRef scene.template;
				in ''{
						Name: ${toJSON scene.name},
						PluginIndex: ${toString ref.pluginIndex},
						TmplIndex: ${toString ref.templateIndex},
					},'';
			groupTemplate = group: ''{
				Name: ${toJSON group.name},
				Description: ${toJSON group.description},
				Config: []byte(`${group.config or ""}`),
				Scenes: []app.SceneTmplRef{
					${concatStringsSep "\n\t\t\t\t\t" (map sceneTemplateRefs group.scenes)}
				},
			},'';
			sceneTemplate = scene: ''{
				Name: ${toJSON scene.name},
				Description: ${toJSON scene.description},
				Config: []byte(`${scene.config or ""}`),
			},'';
			pluginCode = plugin:
				''a.AddPlugin("${plugin.id}", &app.Plugin{
		Name: ${toJSON plugin.name},
		Modules: []*modules.Module{
			${concatStringsSep "\n\t\t\t" (map (m: "&${m.importName}.Descriptor,") plugin.modules)}
		},
		SystemTemplates: []app.SystemTemplate{
			${concatStringsSep "\n\t\t\t" (map systemTemplate (plugin.templates.systems or []))}
		},
		GroupTemplates: []app.GroupTemplate{
			${concatStringsSep "\n\t\t\t" (map groupTemplate (plugin.templates.groups or []))}
		},
		SceneTemplates: []app.SceneTemplate{
			${concatStringsSep "\n\t\t\t" (map sceneTemplate (plugin.templates.scenes or []))}
		},
	})'';
		in concatStringsSep "\n\t" (map pluginCode plugins)
	}
}''