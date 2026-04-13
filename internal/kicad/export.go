package kicad

type ExportedPart struct {
	UUID string

	Reference string

	Value string

	DisplayName string

	Category string

	SymbolLibKey string

	SymbolSrcPath string

	FootprintRef string

	FootprintSrcPath string

	FootprintModuleName string

	Manufacturer string
	MPN          string
	Package      string
	Datasheet    string

	X, Y     float64
	Rotation float64

	InBOM   bool
	OnBoard bool
}

type ExportInput struct {
	ProjectName string

	SchematicUUID string

	Parts []ExportedPart
}

type ExportOutput struct {
	Dir string

	ZipPath string

	Warnings []string
}
