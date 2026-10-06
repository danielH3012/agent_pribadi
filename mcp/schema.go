package mcp

import (
	"encoding/json"
	"reflect"
	"strings"
	"time"
)

var (
	rawMessageType = reflect.TypeOf(json.RawMessage{})
	timeType       = reflect.TypeOf(time.Time{})
)

func schemaOf(t reflect.Type) map[string]any {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch {
	case t == rawMessageType:
		return map[string]any{}
	case t == timeType:
		return map[string]any{"type": "string", "format": "date-time"}
	}
	switch t.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Slice, reflect.Array:
		return map[string]any{"type": "array", "items": schemaOf(t.Elem())}
	case reflect.Map:
		return map[string]any{"type": "object"}
	case reflect.Struct:
		return objectSchema(t)
	}
	return map[string]any{} // interface{} dll: tipe bebas
}

func objectSchema(t reflect.Type) map[string]any {
	props := map[string]any{}
	required := []string{}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		name, opts, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "-" {
			continue
		}
		if name == "" {
			name = f.Name
		}
		s := schemaOf(f.Type)
		if d := f.Tag.Get("desc"); d != "" {
			s["description"] = d
		}
		if e := f.Tag.Get("enum"); e != "" {
			s["enum"] = strings.Split(e, ",")
		}
		props[name] = s
		if !strings.Contains(opts, "omitempty") && f.Type.Kind() != reflect.Ptr {
			required = append(required, name)
		}
	}
	out := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		out["required"] = required
	}
	return out
}
