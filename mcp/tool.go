package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
)

type Tool struct {
	Name        string
	Description string
	Schema      json.RawMessage
	ReadOnly    bool
	Timeout     time.Duration

	describe func(args json.RawMessage) string
	call     func(ctx context.Context, args json.RawMessage) (any, error)
}

type ToolOption func(*Tool)

// ReadOnly menandai tool tidak mengubah apa pun (dilewati oleh AllowReadOnly, dikirim sebagai hint ke client).
func ReadOnly() ToolOption { return func(t *Tool) { t.ReadOnly = true } }

// Timeout membatasi waktu eksekusi. Hanya efektif kalau fungsimu menghormati ctx.
func Timeout(d time.Duration) ToolOption { return func(t *Tool) { t.Timeout = d } }

// DescribeCall mengatur teks di prompt permission (default: argumen JSON mentah).
func DescribeCall(fn func(args json.RawMessage) string) ToolOption {
	return func(t *Tool) { t.describe = fn }
}

func (t Tool) describeArgs(args json.RawMessage) string {
	if t.describe != nil {
		return t.describe(args)
	}
	if len(args) == 0 {
		return "(no arguments)"
	}
	return string(args)
}

// Func membuat tool bertipe. In harus struct; schema dibuat dari field-nya.
func Func[In any](name, desc string, fn func(ctx context.Context, in In) (any, error), opts ...ToolOption) Tool {
	rt := reflect.TypeOf((*In)(nil)).Elem()
	if rt.Kind() != reflect.Struct {
		panic(fmt.Sprintf("mcptools: tool %q: input must be a struct, got %s", name, rt))
	}
	sch := objectSchema(rt)
	raw, err := json.Marshal(sch)
	if err != nil {
		panic(fmt.Sprintf("mcptools: tool %q: schema: %v", name, err))
	}
	required, _ := sch["required"].([]string)

	t := Tool{
		Name: name, Description: desc, Schema: raw,
		call: func(ctx context.Context, args json.RawMessage) (any, error) {
			if err := checkRequired(args, required); err != nil {
				return nil, err
			}
			var in In
			if len(args) > 0 && string(args) != "null" {
				if err := json.Unmarshal(args, &in); err != nil {
					return nil, fmt.Errorf("invalid arguments: %w", err)
				}
			}
			return fn(ctx, in)
		},
	}
	for _, o := range opts {
		o(&t)
	}
	return t
}

// Raw untuk schema yang kamu tulis sendiri dan argumen yang diproses manual.
func Raw(name, desc string, schema json.RawMessage, fn func(ctx context.Context, args json.RawMessage) (any, error), opts ...ToolOption) Tool {
	t := Tool{Name: name, Description: desc, Schema: schema, call: fn}
	for _, o := range opts {
		o(&t)
	}
	return t
}

// checkRequired memberi pesan error yang jelas ke LLM kalau argumen wajib hilang.
func checkRequired(args json.RawMessage, required []string) error {
	if len(required) == 0 {
		return nil
	}
	var m map[string]json.RawMessage
	if len(args) > 0 {
		if err := json.Unmarshal(args, &m); err != nil {
			return fmt.Errorf("invalid arguments: %w", err)
		}
	}
	var missing []string
	for _, k := range required {
		if _, ok := m[k]; !ok {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required argument(s): %s", strings.Join(missing, ", "))
	}
	return nil
}
