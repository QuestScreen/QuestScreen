package data

import (
	"errors"
	"net/http"
	"reflect"
	"strconv"
)

// FontStyle describes possible styles of a font
type FontStyle int

const (
	// Standard is the default font style
	Standard FontStyle = iota
	// Bold is the bold font style
	Bold
	// Italic is the italic font style
	Italic
	// BoldItalic is the bold and italic font style
	BoldItalic
	// NumFontStyles is not a valid FontStyle, but used for iterating.
	NumFontStyles
)

// UnmarshalYAML sets the font style from a YAML scalar
func (fs *FontStyle) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var name string
	if err := unmarshal(&name); err != nil {
		return err
	}
	switch name {
	case "Standard":
		*fs = Standard
	case "Bold":
		*fs = Bold
	case "Italic":
		*fs = Italic
	case "BoldItalic":
		*fs = BoldItalic
	default:
		return errors.New("Unknown font style: " + name)
	}
	return nil
}

// MarshalYAML maps the given font style to a string
func (fs *FontStyle) MarshalYAML() (interface{}, error) {
	switch *fs {
	case Standard:
		return "Standard", nil
	case Bold:
		return "Bold", nil
	case Italic:
		return "Italic", nil
	case BoldItalic:
		return "BoldItalic", nil
	default:
		return nil, errors.New("Unknown font style: " + strconv.Itoa(int(*fs)))
	}
}

// FontSize describes the size of a font.
// Font sizes are relative to the screen size.
type FontSize int

const (
	// SmallFont is the smallest size available
	SmallFont FontSize = iota
	// ContentFont is the size used for content text by default.
	ContentFont
	// MediumFont is a size between ContentFont and HeadingFont.
	MediumFont
	// HeadingFont is the size used for heading text by default.
	HeadingFont
	// LargeFont is a size larger than HeadingFont.
	LargeFont
	// HugeFont is the largest font; usually used for displaying a single word
	// on the screen.
	HugeFont
	// NumFontSizes is not a valid size, but used for iterating
	NumFontSizes
)

// UnmarshalYAML sets the font size from a YAML scalar
func (fs *FontSize) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var name string
	if err := unmarshal(&name); err != nil {
		return err
	}
	switch name {
	case "Small":
		*fs = SmallFont
	case "Content":
		*fs = ContentFont
	case "Medium":
		*fs = MediumFont
	case "Heading":
		*fs = HeadingFont
	case "Large":
		*fs = LargeFont
	case "Huge":
		*fs = HugeFont
	default:
		return errors.New("Unknown font size: " + name)
	}
	return nil
}

// MarshalYAML maps the given font size to a string
func (fs *FontSize) MarshalYAML() (interface{}, error) {
	switch *fs {
	case SmallFont:
		return "Small", nil
	case ContentFont:
		return "Content", nil
	case MediumFont:
		return "Medium", nil
	case HeadingFont:
		return "Heading", nil
	case LargeFont:
		return "Large", nil
	case HugeFont:
		return "Huge", nil
	default:
		return nil, errors.New("Unknown font size: " + strconv.Itoa(int(*fs)))
	}
}

// SelectableFont is used to allow the user to select a font family.
type SelectableFont struct {
	Family      string    `json:"-" yaml:"family"`
	FamilyIndex int32     `yaml:"-" json:"familyIndex"`
	Size        FontSize  `json:"size" yaml:"size"`
	Style       FontStyle `json:"style" yaml:"style"`
}

func (s *StaticData) postProcess(target interface{}, fromYaml bool,
	w http.ResponseWriter) bool {
	switch v := target.(type) {
	case *SelectableFont:
		if fromYaml {
			for i := range s.Fonts {
				if v.Family == s.Fonts[i].Name {
					v.FamilyIndex = int32(i)
					return true
				}
			}
			http.Error(w, "unknown font \""+v.Family+"\"", http.StatusBadRequest)
			return false
		}
		if v.FamilyIndex < 0 || v.FamilyIndex >= int32(len(s.Fonts)) {
			http.Error(w, "font index out of range!", http.StatusBadRequest)
			return false
		}
		v.Family = s.Fonts[v.FamilyIndex].Name
	}
	return true
}

func (s *StaticData) setModuleConfigFieldFrom(target interface{},
	fromYaml bool, data map[string]interface{},
	w http.ResponseWriter) bool {
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
			http.Error(w, "field \""+fieldName+"\" missing!",
				http.StatusBadRequest)
			return false
		}
		field := value.Field(i)

		switch field.Type().Kind() {
		case reflect.Int8, reflect.Int16, reflect.Int, reflect.Int32, reflect.Int64:
			if fromYaml {
				intVal, ok := newVal.(int)
				if !ok {
					http.Error(w, "field \""+fieldName+"\" must be a number!",
						http.StatusBadRequest)
					return false
				}
				field.SetInt(int64(intVal))
			} else {
				floatVal, ok := newVal.(float64)
				if !ok {
					http.Error(w, "field \""+fieldName+"\" must be a number!",
						http.StatusBadRequest)
					return false
				}
				field.SetInt(int64(floatVal))
			}
		case reflect.Uint8, reflect.Uint16, reflect.Uint, reflect.Uint32, reflect.Uint64:
			if fromYaml {
				floatVal, ok := newVal.(float64)
				if !ok {
					http.Error(w, "field \""+fieldName+"\" must be a number!",
						http.StatusBadRequest)
					return false
				}
				field.SetUint(uint64(floatVal))
			} else {
				intVal, ok := newVal.(int)
				if !ok {
					http.Error(w, "field \""+fieldName+"\" must be a number!",
						http.StatusBadRequest)
					return false
				}
				field.SetUint(uint64(intVal))
			}
		case reflect.String:
			stringVal, ok := newVal.(string)
			if !ok {
				http.Error(w, "field \""+fieldName+"\" must be a string!",
					http.StatusBadRequest)
				return false
			}
			field.SetString(stringVal)
		default:
			panic("field \"" + fieldName + "\" has unsupported type " + field.Type().Kind().String())
		}
		delete(data, fieldName)
	}
	for key := range data {
		http.Error(w, "Unknown field \""+key+"\"", http.StatusBadRequest)
		return false
	}

	return s.postProcess(target, fromYaml, w)
}
