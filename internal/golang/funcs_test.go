package golang

import (
	"testing"

	"github.com/kolah/eugene/v2/internal/model"
)

func TestStructTagSortsExtraTags(t *testing.T) {
	s := &model.Schema{Extensions: &model.SchemaExtensions{ExtraTags: map[string]string{
		"validate": "required", "db": "id", "xml": "id", "bson": "_id", "form": "id",
	}}}
	want := "`json:\"id\" bson:\"_id\" db:\"id\" form:\"id\" validate:\"required\" xml:\"id\"`"
	for range 20 {
		if got := StructTag(s, "id", true); got != want {
			t.Fatalf("StructTag() = %s, want %s", got, want)
		}
	}
}
