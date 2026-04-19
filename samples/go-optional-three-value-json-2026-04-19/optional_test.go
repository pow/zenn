package optional

import (
	"encoding/json"
	"testing"
)

type UpdateUserRequest struct {
	Name     Optional[string] `json:"name,omitempty"`
	Nickname Optional[string] `json:"nickname,omitempty"`
}

func TestUnmarshalOmittedFieldIsUnset(t *testing.T) {
	input := `{"name": "Alice"}`
	var req UpdateUserRequest
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if !req.Name.IsSet() {
		t.Error("Name should be set")
	}
	if req.Name.Value != "Alice" {
		t.Errorf("Name.Value = %q, want %q", req.Name.Value, "Alice")
	}
	if !req.Nickname.IsUnset() {
		t.Error("Nickname should be unset when omitted from JSON")
	}
}

func TestUnmarshalExplicitNullIsNull(t *testing.T) {
	input := `{"name": "Alice", "nickname": null}`
	var req UpdateUserRequest
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if !req.Nickname.IsNull() {
		t.Error("Nickname should be null when explicitly set to null")
	}
	if req.Nickname.IsUnset() {
		t.Error("Nickname should not be unset when explicitly set to null")
	}
}

func TestUnmarshalWithValueIsSet(t *testing.T) {
	input := `{"name": "Alice", "nickname": "Bob"}`
	var req UpdateUserRequest
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if !req.Nickname.IsSet() {
		t.Error("Nickname should be set")
	}
	if req.Nickname.Value != "Bob" {
		t.Errorf("Nickname.Value = %q, want %q", req.Nickname.Value, "Bob")
	}
}

func TestMarshalUnsetFieldOmitted(t *testing.T) {
	req := UpdateUserRequest{
		Name: NewValue("Alice"),
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal check error: %v", err)
	}
	if _, ok := m["name"]; !ok {
		t.Error("name should be present in JSON output")
	}
}

func TestMarshalNullFieldOutputsNull(t *testing.T) {
	req := UpdateUserRequest{
		Name:     NewValue("Alice"),
		Nickname: NewNull[string](),
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal check error: %v", err)
	}
	v, ok := m["nickname"]
	if !ok {
		t.Error("nickname should be present in JSON output")
	}
	if v != nil {
		t.Errorf("nickname should be null, got %v", v)
	}
}

func TestApplyToPtrUnsetKeepsCurrent(t *testing.T) {
	current := ptrTo("existing")
	result := ApplyToPtr(NewUnset[string](), current)
	if result != current {
		t.Error("ApplyToPtr with unset should return current pointer")
	}
}

func TestApplyToPtrNullClearsValue(t *testing.T) {
	current := ptrTo("existing")
	result := ApplyToPtr(NewNull[string](), current)
	if result != nil {
		t.Error("ApplyToPtr with null should return nil")
	}
}

func TestApplyToPtrValueSetsNew(t *testing.T) {
	current := ptrTo("existing")
	result := ApplyToPtr(NewValue("updated"), current)
	if result == nil {
		t.Fatal("ApplyToPtr with value should not return nil")
	}
	if *result != "updated" {
		t.Errorf("ApplyToPtr result = %q, want %q", *result, "updated")
	}
	if result == current {
		t.Error("ApplyToPtr should return a new pointer, not reuse current")
	}
}

func ptrTo[T any](v T) *T { return &v }
