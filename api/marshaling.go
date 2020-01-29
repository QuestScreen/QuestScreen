package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

// ReceiveData loads JSON input into a target object and wraps errors in
// BadRequest objects.
func ReceiveData(input []byte, target interface{}) *BadRequest {
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return &BadRequest{"error in JSON structure", err}
	}
	return nil
}

// ValidatedInt can be used to load an integer value that must be in a
// specified range.
type ValidatedInt struct {
	// data is loaded into this
	Value int
	// inclusive required range
	Min, Max int
}

// UnmarshalJSON loads the given JSON input as int value and on success checks
// whether the loaded value is inside the required range.
func (vi *ValidatedInt) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &vi.Value); err != nil {
		return err
	}
	if vi.Value < vi.Min || vi.Value > vi.Max {
		return fmt.Errorf("value outside of allowed range [%d..%d]",
			vi.Min, vi.Max)
	}
	return nil
}

// ValidatedString can be used to load a string value whose length must be in a
// specified range.
type ValidatedString struct {
	// data is loaded into this
	Value string
	// ignored if -1
	MinLen, MaxLen int
}

// UnmarshalJSON loads the given JSON input as string value and on success
// checks whether the loaded value's length is in the required range.
func (vs *ValidatedString) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &vs.Value); err != nil {
		return err
	}
	l := len(vs.Value)
	if (vs.MinLen >= 0 && l < vs.MinLen) || (vs.MaxLen >= 0 && l > vs.MaxLen) {
		var msg string
		if vs.MinLen == 1 && vs.MaxLen == -1 {
			msg = "string must not be empty"
		} else if vs.MaxLen == -1 {
			msg = fmt.Sprintf("string must be at least %d characters long",
				vs.MinLen)
		} else if vs.MinLen == -1 {
			msg = fmt.Sprintf("string must not be longer than %d characters",
				vs.MaxLen)
		} else {
			msg = fmt.Sprintf("string must be between %d and %d characters long",
				vs.MinLen, vs.MaxLen)
		}
		return errors.New(msg)
	}
	return nil
}

// ValidatedStruct can be used to load a struct value for which each field must
// exist in the input.
type ValidatedStruct struct {
	Value interface{}
}

var knownFieldMappings map[reflect.Type]map[string]int = make(map[reflect.Type]map[string]int)

// UnmarshalJSON loads the given JSON input as object and assigns each value to
// the target's field with the same name (honoring a field's json tag). It
// requires each field to be given a value.
func (vs *ValidatedStruct) UnmarshalJSON(data []byte) error {
	structType := reflect.TypeOf(vs.Value)
	structValue := reflect.ValueOf(vs.Value)
	if structType.Kind() != reflect.Ptr {
		panic("non-pointer value given to ValidatedStruct")
	}
	for structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
		structValue = structValue.Elem()
	}
	if structType.Kind() != reflect.Struct {
		panic("ValidatedStruct used on a non-struct value")
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	found := make([]bool, structType.NumField())
	fieldNameMap, ok := knownFieldMappings[structType]
	if !ok {
		fieldNameMap = make(map[string]int)
		for i := 0; i < structType.NumField(); i++ {
			tagVal, ok := structType.Field(i).Tag.Lookup("json")
			fieldName := structType.Field(i).Name
			if ok {
				if tagVal == "-" {
					continue
				}
				fieldName = tagVal
			}
			fieldNameMap[fieldName] = i
		}
		knownFieldMappings[structType] = fieldNameMap
	}

	for k, v := range raw {
		index, ok := fieldNameMap[k]
		if !ok {
			return fmt.Errorf("unknown field %s.\"%s\"",
				structType.Name(), k)
		}
		if found[index] {
			return fmt.Errorf("duplicate value for field %s.\"%s\"",
				structType.Name(), k)
		}
		found[index] = true
		if err := json.Unmarshal(v,
			structValue.Field(index).Addr().Interface()); err != nil {
			return fmt.Errorf("in %s.%s: %s", structType.Name(),
				structType.Field(index).Name, err.Error())
		}
	}
	for i := range found {
		if !found[i] {
			return fmt.Errorf("missing value for %s.%s",
				structType.Name(), structType.Field(i).Name)
		}
	}
	return nil
}

// ValidatedSlice takes a pointer to a slice as data and deserializes
// a JSON value into it, checking the required bounds.
type ValidatedSlice struct {
	Data     interface{}
	MinItems int
	MaxItems int
}

// UnmarshalJSON loads JSON input into a slice, validating its length
func (vs *ValidatedSlice) UnmarshalJSON(data []byte) error {
	sliceType := reflect.TypeOf(vs.Data)
	sliceValue := reflect.ValueOf(vs.Data)
	if sliceType.Kind() != reflect.Ptr {
		panic("gave non-pointer data to ValidatedSlice")
	}
	if err := json.Unmarshal(data, vs.Data); err != nil {
		return err
	}
	sliceType = sliceType.Elem()
	sliceValue = sliceValue.Elem()
	if sliceType.Kind() != reflect.Slice {
		panic("ValidatedSlice used on a non-ptr-to-slice value")
	}

	if sliceValue.Len() < vs.MinItems || sliceValue.Len() > vs.MaxItems {
		return fmt.Errorf("array length outside of supported length [%d..%d]",
			vs.MinItems, vs.MaxItems)
	}
	return nil
}
