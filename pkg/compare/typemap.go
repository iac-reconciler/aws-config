package compare

import (
	_ "embed"
	"encoding/json"

	log "github.com/sirupsen/logrus"
)

// awsTerraformToConfigTypeMap maps types from terraform to config types
//
//go:embed typemap.json
var awsTerraformToConfigTypeMapJSON []byte

var (
	awsTerraformToConfigTypeMap map[string]string
	awsConfigToTerraformypeMap  map[string]string
)

func init() {
	awsTerraformToConfigTypeMap = make(map[string]string)
	awsConfigToTerraformypeMap = make(map[string]string)
	if err := json.Unmarshal(awsTerraformToConfigTypeMapJSON, &awsTerraformToConfigTypeMap); err != nil {
		log.Fatalf("unable to unmarshal typemap.json: %v", err)
	}
	for k, v := range awsTerraformToConfigTypeMap {
		awsConfigToTerraformypeMap[v] = k
	}
}
