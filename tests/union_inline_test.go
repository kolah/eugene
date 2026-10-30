package tests

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	union "github.com/kolah/eugene/v2/tests/generated/types_union_inline"
)

func TestUnionInlineAccessors(t *testing.T) {
	var msg union.Message
	require.NoError(t, json.Unmarshal([]byte(`{"context":{"list":["a"],"text":"a"}}`), &msg))

	list := msg.Context["list"]
	gotList, err := list.AsStringList()
	require.NoError(t, err)
	require.Equal(t, []string{"a"}, *gotList)

	text := msg.Context["text"]
	gotText, err := text.AsString()
	require.NoError(t, err)
	require.Equal(t, "a", *gotText)
}

func TestUnionInlineObjectsGetDistinctTypes(t *testing.T) {
	var v union.Value
	require.NoError(t, json.Unmarshal([]byte(`{"a":"x","b":1}`), &v))

	first, err := v.AsValueVariant()
	require.NoError(t, err)
	require.Equal(t, "x", *first.A)

	second, err := v.AsValueVariant2()
	require.NoError(t, err)
	require.Equal(t, 1, *second.B)
}

func TestUnionRefAccessorKeepsSchemaName(t *testing.T) {
	var v union.Value
	require.NoError(t, json.Unmarshal([]byte(`["a"]`), &v))

	got, err := v.AsList()
	require.NoError(t, err)
	require.Equal(t, union.List{"a"}, *got)
}

func TestUnionPreservesRawValue(t *testing.T) {
	const date = `"2024-01-02"`

	var v union.Value
	require.NoError(t, json.Unmarshal([]byte(date), &v))

	out, err := json.Marshal(v)
	require.NoError(t, err)
	require.JSONEq(t, date, string(out))
}

func TestUnionGoNameAccessor(t *testing.T) {
	var w union.Window
	require.NoError(t, json.Unmarshal([]byte(`"2024-01-02T03:04:05Z"`), &w))

	got, err := w.AsStartsAt()
	require.NoError(t, err)
	require.Equal(t, 2024, got.Year())
}
