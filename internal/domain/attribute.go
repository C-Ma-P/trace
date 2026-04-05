package domain

type ValueType string

const (
	ValueTypeText   ValueType = "text"
	ValueTypeNumber ValueType = "number"
	ValueTypeBool   ValueType = "bool"
)

type AttributeDefinition struct {
	Key         string    `db:"key"`
	Category    Category  `db:"category"`
	ValueType   ValueType `db:"value_type"`
	DisplayName string    `db:"display_name"`
	Unit        *string   `db:"unit"`
}

type AttributeValue struct {
	Key       string
	ValueType ValueType
	Text      *string
	Number    *float64
	Bool      *bool
	Unit      string
}
