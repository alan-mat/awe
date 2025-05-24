package api

type DataType string

const (
	TypeString  DataType = "STRING"
	TypeNumber  DataType = "NUMBER"
	TypeInteger DataType = "INTEGER"
	TypeBoolean DataType = "BOOLEAN"
	TypeArray   DataType = "ARRAY"
	TypeObject  DataType = "OBJECT"
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
