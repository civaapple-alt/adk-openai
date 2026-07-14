package adkopenai

import (
	"encoding/json"
	"fmt"

	"google.golang.org/genai"
)

// normalizeJSONSchema converts ADK's arbitrary JSON Schema representation
// (commonly *jsonschema.Schema) into a map[string]any.
func normalizeJSONSchema(schema any) (map[string]any, error) {
	raw, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("marshal JSON schema: %w", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("unmarshal JSON schema: %w", err)
	}
	if out == nil {
		return nil, fmt.Errorf("JSON schema must be an object")
	}
	return out, nil
}

func convertSchema(schema *genai.Schema) (map[string]any, error) {
	if schema == nil {
		return map[string]any{"type": "object", "properties": map[string]any{}}, nil
	}
	result := map[string]any{}
	if schema.Type != genai.TypeUnspecified {
		result["type"] = schemaType(schema.Type)
	}
	if schema.Title != "" {
		result["title"] = schema.Title
	}
	if schema.Description != "" {
		result["description"] = schema.Description
	}
	if schema.Format != "" {
		result["format"] = schema.Format
	}
	if schema.Pattern != "" {
		result["pattern"] = schema.Pattern
	}
	if schema.Nullable != nil {
		result["nullable"] = *schema.Nullable
	}
	if schema.Default != nil {
		result["default"] = schema.Default
	}
	if schema.Example != nil {
		result["example"] = schema.Example
	}
	if schema.Minimum != nil {
		result["minimum"] = *schema.Minimum
	}
	if schema.Maximum != nil {
		result["maximum"] = *schema.Maximum
	}
	if schema.MinLength != nil {
		result["minLength"] = *schema.MinLength
	}
	if schema.MaxLength != nil {
		result["maxLength"] = *schema.MaxLength
	}
	if schema.MinItems != nil {
		result["minItems"] = *schema.MinItems
	}
	if schema.MaxItems != nil {
		result["maxItems"] = *schema.MaxItems
	}
	if schema.MinProperties != nil {
		result["minProperties"] = *schema.MinProperties
	}
	if schema.MaxProperties != nil {
		result["maxProperties"] = *schema.MaxProperties
	}
	if len(schema.Enum) > 0 {
		result["enum"] = schema.Enum
	}
	if len(schema.Required) > 0 {
		result["required"] = schema.Required
	}
	if len(schema.PropertyOrdering) > 0 {
		result["propertyOrdering"] = schema.PropertyOrdering
	}
	if len(schema.AnyOf) > 0 {
		anyOf := make([]any, 0, len(schema.AnyOf))
		for _, item := range schema.AnyOf {
			converted, err := convertSchema(item)
			if err != nil {
				return nil, err
			}
			anyOf = append(anyOf, converted)
		}
		result["anyOf"] = anyOf
	}
	if len(schema.Properties) > 0 {
		props := make(map[string]any, len(schema.Properties))
		for name, prop := range schema.Properties {
			converted, err := convertSchema(prop)
			if err != nil {
				return nil, err
			}
			props[name] = converted
		}
		result["properties"] = props
	}
	if schema.Items != nil {
		items, err := convertSchema(schema.Items)
		if err != nil {
			return nil, err
		}
		result["items"] = items
	}
	return result, nil
}

func schemaType(t genai.Type) string {
	switch t {
	case genai.TypeString:
		return "string"
	case genai.TypeNumber:
		return "number"
	case genai.TypeInteger:
		return "integer"
	case genai.TypeBoolean:
		return "boolean"
	case genai.TypeArray:
		return "array"
	case genai.TypeObject:
		return "object"
	default:
		return "string"
	}
}

func functionParametersFromDecl(decl *genai.FunctionDeclaration) (map[string]any, error) {
	if decl == nil {
		return map[string]any{"type": "object", "properties": map[string]any{}}, nil
	}
	if decl.ParametersJsonSchema != nil {
		return normalizeJSONSchema(decl.ParametersJsonSchema)
	}
	if decl.Parameters != nil {
		return convertSchema(decl.Parameters)
	}
	return map[string]any{"type": "object", "properties": map[string]any{}}, nil
}
