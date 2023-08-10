package load

const (
	TerraformManaged = "managed"
)

type TerraformState struct {
	Version          int
	TerraformVersion string
	Serial           int
	Lineage          string
	Outputs          map[string]interface{}
	Resources        []Resource
}

type Resource struct {
	Module    string
	Mode      string
	Type      string
	Name      string
	Provider  string
	Instances []Instance
}

type Instance struct {
	SchemaVersion int
	Attributes    map[string]interface{}
	Private       string
	IndexKey      string
}
