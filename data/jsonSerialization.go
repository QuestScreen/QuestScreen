package data

import (
	"encoding/json"
	"errors"
	"reflect"
)

type jsonConfigItem struct {
	Value   interface{} `json:"value"`
	Default interface{} `json:"default"`
}

type jsonModuleConfig []jsonConfigItem

func (s *StaticData) loadJSONModuleConfigInto(target interface{},
	raw []interface{}) error {
	targetModule := reflect.ValueOf(target).Elem()
	targetModuleType := targetModule.Type()
	for i := 0; i < targetModuleType.NumField(); i++ {
		wasNil := false
		if targetModule.Field(i).IsNil() {
			targetModule.Field(i).Set(reflect.New(targetModuleType.Field(i).Type.Elem()))
			wasNil = true
		}
		targetSetting := targetModule.Field(i).Interface()
		inModuleConfig := raw[i].(map[string]interface{})

		if err := s.setModuleConfigFieldFrom(targetSetting, false, inModuleConfig); err != nil {
			if wasNil {
				targetModule.Field(i).Set(reflect.Zero(targetModuleType.Field(i).Type))
			}
			return err
		}
	}
	return nil
}

func (s *StaticData) loadJSONModuleConfigs(jsonInput []byte,
	targetConfigs []interface{}) error {
	var raw []interface{}
	if err := json.Unmarshal(jsonInput, &raw); err != nil {
		return err
	}
	for i := 0; i < len(targetConfigs); i++ {
		moduleFields, ok := raw[i].([]interface{})
		if !ok {
			return errors.New("data for module\"" + s.items.ItemAt(i).Name() + " is not a JSON array!")
		}
		conf := targetConfigs[i]
		if err := s.loadJSONModuleConfigInto(conf, moduleFields); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) buildModuleConfigJSON(config []interface{}) []jsonModuleConfig {
	ret := make([]jsonModuleConfig, 0, s.items.NumItems())
	for i := 0; i < s.items.NumItems(); i++ {
		item := s.items.ItemAt(i)
		itemConfig := config[i]
		itemValue := reflect.ValueOf(itemConfig).Elem()
		for ; itemValue.Kind() == reflect.Interface ||
			itemValue.Kind() == reflect.Ptr; itemValue = itemValue.Elem() {
		}
		curConfig := item.GetConfig()
		jsonConfig := make(jsonModuleConfig, 0, itemValue.NumField())
		curValue := reflect.ValueOf(curConfig).Elem()
		for j := 0; j < itemValue.NumField(); j++ {
			jsonConfig = append(jsonConfig, jsonConfigItem{
				Value:   itemValue.Field(j).Interface(),
				Default: curValue.Field(j).Interface()})
		}
		ret = append(ret, jsonConfig)
	}
	return ret
}
