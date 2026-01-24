package golang

import (
	"sort"
	"strings"
)

// EnumUsage records where an enum is used in the spec.
type EnumUsage struct {
	FieldName  string
	ParentName string
	Values     []string
	ValuesKey  string
}

// EnumRegistry collects all enum usages and resolves canonical names.
// It ensures stable, context-aware naming based on field names and values.
type EnumRegistry struct {
	usages         []EnumUsage
	valueToName    map[string]string
	nameToValues   map[string]string
	generatedTypes map[string]ResolvedType
}

// NewEnumRegistry creates a new EnumRegistry.
func NewEnumRegistry() *EnumRegistry {
	return &EnumRegistry{
		valueToName:    make(map[string]string),
		nameToValues:   make(map[string]string),
		generatedTypes: make(map[string]ResolvedType),
	}
}

// CollectEnum records an enum usage for later name resolution.
func (r *EnumRegistry) CollectEnum(fieldName, parentName string, values []any) {
	strs := toStringSlice(values)
	r.usages = append(r.usages, EnumUsage{
		FieldName:  fieldName,
		ParentName: parentName,
		Values:     strs,
		ValuesKey:  canonicalKey(strs),
	})
}

// ResolveNames processes all collected usages and assigns canonical names.
// Names are derived from field names with collision handling based on values.
func (r *EnumRegistry) ResolveNames() {
	// Group usages by values
	groups := make(map[string][]EnumUsage)
	for _, u := range r.usages {
		groups[u.ValuesKey] = append(groups[u.ValuesKey], u)
	}

	// Sort keys for deterministic ordering
	var keys []string
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, valuesKey := range keys {
		usages := groups[valuesKey]
		name := r.determineName(usages, valuesKey)
		r.valueToName[valuesKey] = name
		r.nameToValues[name] = valuesKey
	}
}

func (r *EnumRegistry) determineName(usages []EnumUsage, valuesKey string) string {
	// Count field name frequency
	fieldCounts := make(map[string]int)
	for _, u := range usages {
		fieldCounts[u.FieldName]++
	}

	// Find most common field name (alphabetical tie-break)
	var bestField string
	var bestCount int
	for field, count := range fieldCounts {
		if count > bestCount || (count == bestCount && field < bestField) {
			bestField = field
			bestCount = count
		}
	}

	baseName := PascalCase(bestField)

	// Check for collision with different values
	if existingKey, taken := r.nameToValues[baseName]; taken && existingKey != valuesKey {
		// Add suffix from sorted values
		suffix := valueSuffix(usages[0].Values)
		return baseName + suffix
	}

	return baseName
}

// GetCanonicalName returns the predetermined name for enum values.
func (r *EnumRegistry) GetCanonicalName(values []any) (string, bool) {
	key := canonicalKey(toStringSlice(values))
	name, ok := r.valueToName[key]
	return name, ok
}

// MarkGenerated records that a type has been generated.
func (r *EnumRegistry) MarkGenerated(name string, rt ResolvedType) {
	r.generatedTypes[name] = rt
}

// IsGenerated checks if a type has already been generated.
func (r *EnumRegistry) IsGenerated(name string) bool {
	_, ok := r.generatedTypes[name]
	return ok
}

// GetGeneratedType returns the generated type if it exists.
func (r *EnumRegistry) GetGeneratedType(name string) (ResolvedType, bool) {
	rt, ok := r.generatedTypes[name]
	return rt, ok
}

func canonicalKey(values []string) string {
	sorted := make([]string, len(values))
	copy(sorted, values)
	sort.Strings(sorted)
	return strings.Join(sorted, "|")
}

func toStringSlice(values []any) []string {
	strs := make([]string, 0, len(values))
	for _, v := range values {
		if s, ok := v.(string); ok {
			strs = append(strs, s)
		}
	}
	return strs
}

func valueSuffix(values []string) string {
	if len(values) == 0 {
		return ""
	}
	sorted := make([]string, len(values))
	copy(sorted, values)
	sort.Strings(sorted)
	// Use first value (or first two if available)
	if len(sorted) >= 2 {
		return PascalCase(sorted[0]) + PascalCase(sorted[1])
	}
	return PascalCase(sorted[0])
}
