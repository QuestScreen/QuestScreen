package api

import (
	"fmt"

	"github.com/veandco/go-sdl2/ttf"
	"gopkg.in/yaml.v3"
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

// StyledFont describes a font family member with a certain style.
// This style is available for all FontSizes.
type StyledFont interface {
	// Font searches for a loaded ttf.Font and if none exists, loads it.
	// This func may only be called in the OpenGL thread.
	Font(size FontSize) *ttf.Font
}

// FontFamily describes a family of fonts. The family may have any number,
// but at least one, FontStyle available.
type FontFamily interface {
	// Name of this font family.
	Name() string
	// Styled returns a StyledFont that matches the requested style as good as
	// possible. If the requested style is not available, a style close to it
	// is selected.
	Styled(style FontStyle) StyledFont
}

// UnmarshalYAML sets the font style from a YAML scalar
func (fs *FontStyle) UnmarshalYAML(value *yaml.Node) error {
	var name string
	if err := value.Decode(&name); err != nil {
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
		return fmt.Errorf("Unknown font style: %s", name)
	}
	return nil
}

// MarshalYAML maps the given font style to a string
func (fs FontStyle) MarshalYAML() (interface{}, error) {
	switch fs {
	case Standard:
		return "Standard", nil
	case Bold:
		return "Bold", nil
	case Italic:
		return "Italic", nil
	case BoldItalic:
		return "BoldItalic", nil
	default:
		return nil, fmt.Errorf("Unknown font style: %v", fs)
	}
}

// UnmarshalYAML sets the font size from a YAML scalar
func (fs *FontSize) UnmarshalYAML(value *yaml.Node) error {
	var name string
	if err := value.Decode(&name); err != nil {
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
		return fmt.Errorf("Unknown font size: %s", name)
	}
	return nil
}

// MarshalYAML maps the given font size to a string
func (fs FontSize) MarshalYAML() (interface{}, error) {
	switch fs {
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
		return nil, fmt.Errorf("Unknown font size: %v", fs)
	}
}
