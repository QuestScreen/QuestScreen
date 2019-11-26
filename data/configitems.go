package data

import (
	"errors"
	"github.com/flyx/pnpscreen/api"
	"reflect"
)

// setModuleConfigFieldFrom assigns the given data to the given target.
// data is expected to be deserialized from a JSON or YAML structure that
// matches the target type's structure.
func (c *Config) setModuleConfigFieldFrom(target interface{},
	fromYaml bool, data map[string]interface{}) error {
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

		newVal, ok := data[fieldName]
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
		delete(data, fieldName)
	}
	for key := range data {
		return errors.New("Unknown field \"" + key + "\"")
	}

	return target.(api.ConfigItem).PostLoad(c.owner, fromYaml)
}
