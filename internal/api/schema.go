package api

import "encoding/json"

type DataType string

const (
	TypeString  DataType = "string"
	TypeNumber  DataType = "number"
	TypeInteger DataType = "integer"
	TypeBoolean DataType = "boolean"
	TypeArray   DataType = "array"
	TypeObject  DataType = "object"
)

// Schema is an incomplete OpenAPI 3.0 schema object
type Schema struct {
	Description string             `json:"description,omitempty"`
	Items       *Schema            `json:"items,omitempty"`
	Properties  map[string]*Schema `json:"properties,omitempty"`
	Required    []string           `json:"required,omitempty"`
	Title       string             `json:"title,omitempty"`
	Type        DataType           `json:"type,omitempty"`
}

func (s Schema) MarshalJSON() ([]byte, error) {
	type Alias Schema
	return json.Marshal(&struct {
		Alias
	}{
		Alias: (Alias)(s),
	})
}

func (s *Schema) UnmarshalJSON(data []byte) error {
	type Alias Schema
	aux := &struct {
		Alias
	}{}

	if err := json.Unmarshal(data, &aux.Alias); err != nil {
		return err
	}

	*s = Schema(aux.Alias)

	return nil
}
