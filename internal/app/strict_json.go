package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
)

// decodeStrictJSON rejects schema drift and ambiguous duplicate object keys
// before decoding a controller-routing artifact.
func decodeStrictJSON(data []byte, destination any) error {
	keys := json.NewDecoder(bytes.NewReader(data))
	if err := rejectDuplicateKeys(keys); err != nil {
		return err
	}
	if _, err := keys.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}
		return err
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}
		return err
	}
	return nil
}

// decodeStrictJSONExactRequired is for untrusted, versioned artifact
// boundaries where every non-omitempty member is required. encoding/json
// otherwise accepts case-insensitive aliases and cannot distinguish an
// omitted zero-valid bool from an explicit false value.
func decodeStrictJSONExactRequired(data []byte, destination any) error {
	if err := decodeStrictJSON(data, destination); err != nil {
		return err
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	typeOf := reflect.TypeOf(destination)
	if typeOf == nil || typeOf.Kind() != reflect.Pointer || typeOf.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("exact JSON destination must point to a struct")
	}
	return validateExactJSONMembers(value, typeOf.Elem(), "")
}

func validateExactJSONMembers(value any, schema reflect.Type, path string) error {
	for schema.Kind() == reflect.Pointer {
		schema = schema.Elem()
	}
	if value == nil {
		return fmt.Errorf("field %q must be a JSON %s, not null", path, jsonTypeName(schema))
	}
	switch schema.Kind() {
	case reflect.Struct:
		object, ok := value.(map[string]any)
		if !ok {
			return nil // decodeStrictJSON already reports JSON type mismatches.
		}
		fields := make(map[string]reflect.Type)
		required := make(map[string]bool)
		for index := 0; index < schema.NumField(); index++ {
			field := schema.Field(index)
			if field.PkgPath != "" {
				continue
			}
			tag := field.Tag.Get("json")
			parts := strings.Split(tag, ",")
			name := parts[0]
			if name == "-" {
				continue
			}
			if name == "" {
				name = field.Name
			}
			fields[name] = field.Type
			optional := false
			for _, option := range parts[1:] {
				optional = optional || option == "omitempty"
			}
			required[name] = !optional
		}
		for name := range object {
			if _, exists := fields[name]; !exists {
				return fmt.Errorf("unknown field %q", jsonPath(path, name))
			}
		}
		for name, fieldType := range fields {
			child, exists := object[name]
			if !exists {
				if required[name] {
					return fmt.Errorf("missing required field %q", jsonPath(path, name))
				}
				continue
			}
			if err := validateExactJSONMembers(child, fieldType, jsonPath(path, name)); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		items, ok := value.([]any)
		if !ok {
			return nil
		}
		for index, item := range items {
			if err := validateExactJSONMembers(item, schema.Elem(), fmt.Sprintf("%s[%d]", path, index)); err != nil {
				return err
			}
		}
	}
	return nil
}

func jsonTypeName(schema reflect.Type) string {
	for schema.Kind() == reflect.Pointer {
		schema = schema.Elem()
	}
	switch schema.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Struct, reflect.Map:
		return "object"
	default:
		return "value"
	}
}

func jsonPath(parent, name string) string {
	if parent == "" {
		return name
	}
	return parent + "." + name
}

func rejectDuplicateKeys(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, compound := token.(json.Delim)
	if !compound {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("invalid JSON object key")
			}
			if _, exists := seen[key]; exists {
				return fmt.Errorf("duplicate JSON key %q", key)
			}
			seen[key] = struct{}{}
			if err := rejectDuplicateKeys(decoder); err != nil {
				return err
			}
		}
	case '[':
		for decoder.More() {
			if err := rejectDuplicateKeys(decoder); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}
	closing, err := decoder.Token()
	if err != nil {
		return err
	}
	want := json.Delim('}')
	if delimiter == '[' {
		want = ']'
	}
	if closing != want {
		return fmt.Errorf("unexpected JSON delimiter %q", closing)
	}
	return nil
}
