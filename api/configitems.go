package api

import (
	"errors"
	"fmt"
	"reflect"
)

// this file contains types that may be used as fields in a module's
// configuration struct.

// ConfigItem describes an item in a module's configuration.
// A ConfigItem's public fields will be loaded from YAML structure automatically
// via reflection, and JSON serialization will also be done via reflection.
// you may use the tags `json:` and `yaml:` on those fields as documented in
// the json and yaml.v3 packages.
type ConfigItem interface {
	LoadFrom(subtree interface{}, env Environment, fromYaml bool) error
	Serialize(env Environment, toYAML bool) interface{}
}

// SelectableFont is used to allow the user to select a font family.
type SelectableFont struct {
	FamilyIndex int       `json:"familyIndex"`
	Size        FontSize  `json:"size"`
	Style       FontStyle `json:"style"`
}

type yamlSelectableFont struct {
	Family string    `yaml:"family"`
	Size   FontSize  `yaml:"size"`
	Style  FontStyle `yaml:"style"`
}

// LoadFrom loads values from a JSON/YAML subtree
func (sf *SelectableFont) LoadFrom(subtree interface{}, env Environment,
	fromYAML bool) error {
	fonts := env.FontCatalog()
	if fromYAML {
		var tmp yamlSelectableFont
		if err := SubtreeToConfigItem(subtree.(map[string]interface{}), &tmp, true); err != nil {
			return err
		}
		sf.Size = tmp.Size
		sf.Style = tmp.Style
		for i := range fonts {
			if tmp.Family == fonts[i].Name() {
				sf.FamilyIndex = i
				return nil
			}
		}
		return fmt.Errorf("unknown font \"%s\"", tmp.Family)
	}

	if err := SubtreeToConfigItem(subtree.(map[string]interface{}), sf, false); err != nil {
		return err
	}
	if sf.FamilyIndex < 0 || sf.FamilyIndex >= len(fonts) {
		return errors.New("font index out of range")
	}
	return nil
}

// Serialize returns the object itself for JSON and an object with the family
// name instead of its index for YAML
func (sf *SelectableFont) Serialize(env Environment, toYAML bool) interface{} {
	if toYAML {
		return &yamlSelectableFont{
			Family: env.FontCatalog()[sf.FamilyIndex].Name(),
			Size:   sf.Size,
			Style:  sf.Style,
		}
	}
	return sf
}

// SubtreeToConfigItem fills the public fields of the given target with the
// values contained in the given subtree.
//
// This function offers simple deserialization based on the type's layout.
// Use it if you don't need anything fancy. If target is directly the
// ConfigItem, you will be able to implement Serialize on the
// ConfigItem by simply returning the item itself.
//
// This func honors the `yaml:` and `json:` tags on the target type's fields.
func SubtreeToConfigItem(subtree interface{}, target interface{},
	fromYaml bool) error {
	properSubtree, ok := subtree.(map[string]interface{})
	// this is a fix for a problem in the yaml lib that
	// leads to yaml giving the type map[interface{}]interface{}.
	if !ok {
		raw, ok := subtree.(map[interface{}]interface{})
		if !ok {
			return fmt.Errorf(
				"cannot load values for %s from a subtree which is not a map",
				reflect.TypeOf(target).String())
		}
		properSubtree = make(map[string]interface{})
		for key, value := range raw {
			stringKey, ok := key.(string)
			if !ok {
				return fmt.Errorf(
					"cannot load value for %s from a subtree map with non-string keys",
					reflect.TypeOf(target).String())
			}
			properSubtree[stringKey] = value
		}
	}

	settingType := reflect.TypeOf(target)
	value := reflect.ValueOf(target)
	for settingType.Kind() == reflect.Interface ||
		settingType.Kind() == reflect.Ptr {
		settingType = settingType.Elem()
		value = value.Elem()
	}
	if settingType.Kind() != reflect.Struct || value.Kind() != reflect.Struct {
		panic("setting type is not a struct!")
	}
	for i := 0; i < settingType.NumField(); i++ {
		tagName := "json"
		if fromYaml {
			tagName = "yaml"
		}
		tagVal, ok := settingType.Field(i).Tag.Lookup(tagName)
		fieldName := settingType.Field(i).Name
		if ok {
			if tagVal == "-" {
				continue
			}
			fieldName = tagVal
		}

		newVal, ok := properSubtree[fieldName]
		if !ok {
			return errors.New("field \"" + fieldName + "\" missing!")
		}
		field := value.Field(i)

		switch field.Type().Kind() {
		case reflect.Int8, reflect.Int16, reflect.Int, reflect.Int32, reflect.Int64:
			if fromYaml {
				intVal, ok := newVal.(int)
				if !ok {
					return errors.New("field \"" + fieldName + "\" must be a number!")
				}
				field.SetInt(int64(intVal))
			} else {
				floatVal, ok := newVal.(float64)
				if !ok {
					return errors.New("field \"" + fieldName + "\" must be a number!")
				}
				field.SetInt(int64(floatVal))
			}
		case reflect.Uint8, reflect.Uint16, reflect.Uint, reflect.Uint32, reflect.Uint64:
			if fromYaml {
				floatVal, ok := newVal.(float64)
				if !ok {
					return errors.New("field \"" + fieldName + "\" must be a number!")
				}
				field.SetUint(uint64(floatVal))
			} else {
				intVal, ok := newVal.(int)
				if !ok {
					return errors.New("field \"" + fieldName + "\" must be a number!")
				}
				field.SetUint(uint64(intVal))
			}
		case reflect.String:
			stringVal, ok := newVal.(string)
			if !ok {
				return errors.New("field \"" + fieldName + "\" must be a string!")
			}
			field.SetString(stringVal)
		default:
			panic("field \"" + fieldName + "\" has unsupported type " + field.Type().Kind().String())
		}
		delete(properSubtree, fieldName)
	}
	for key := range properSubtree {
		return errors.New("Unknown field \"" + key + "\"")
	}
	return nil
}
