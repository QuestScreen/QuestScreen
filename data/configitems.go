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
	Family      string `json:"-"`
	FamilyIndex int32  `yaml:"-"`
	Size        FontSize
	Style       FontStyle
}

func setInt(field *int, name string, json map[string]interface{},
	w http.ResponseWriter) bool {
	val, ok := json[name]
	if !ok {
		http.Error(w, "field \""+name+"\" missing!", http.StatusBadRequest)
		return false
	}
	floatVal, ok := val.(float64)
	if ok {
		*field = int(floatVal)
		return true
	}
	http.Error(w, "field \""+name+"\" must be a number!", http.StatusBadRequest)
	return false
}

func setInt32(field *int32, name string, json map[string]interface{},
	w http.ResponseWriter) bool {
	val, ok := json[name]
	if !ok {
		http.Error(w, "field \""+name+"\" missing!", http.StatusBadRequest)
		return false
	}
	floatVal, ok := val.(float64)
	if ok {
		*field = int32(floatVal)
		return true
	}
	http.Error(w, "field \""+name+"\" must be a number!", http.StatusBadRequest)
	return false
}

func (s *StaticData) setFromJSON(target interface{}, json map[string]interface{},
	w http.ResponseWriter) bool {
	// TODO: change code: set via reflection, then do post-processing based on
	// the actual type. ensure all fields are set and no unknown fields are present.

	switch v := target.(type) {
	case *SelectableFont:
		var found [3]bool
		for key, value := range json {
			switch key {
			case "familyIndex":
				if found[0] {
					http.Error(w, "duplicate key: familyIndex", http.StatusBadRequest)
					return false
				}
				found[0] = true

			}
		}

		if !setInt32(&v.FamilyIndex, "FamilyIndex", json, w) {
			return false
		}
		if v.FamilyIndex < 0 || v.FamilyIndex >= int32(len(s.Fonts)) {
			http.Error(w, "font index out of range!", http.StatusBadRequest)
			return false
		}
		v.Family = s.Fonts[v.FamilyIndex].Name
		var sizeInt int
		if !setInt(&sizeInt, "Size", json, w) {
			return false
		}
		v.Size = FontSize(sizeInt)
		var styleInt int
		if !setInt(&styleInt, "Style", json, w) {
			return false
		}
		v.Style = FontStyle(styleInt)
	default:
		panic("unknown type: " + reflect.TypeOf(target).Name())
	}
	return true
}
