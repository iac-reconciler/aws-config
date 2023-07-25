package compare

import (
	"fmt"

	"github.com/iac-reconciler/tf-aws-config/pkg/load"
	log "github.com/sirupsen/logrus"
)

type Reconciled struct {
}

// Reconcile reconcile the snapshot and tfstates.
// Not yet implemented, so returns an empty struct
func Reconcile(snapshot load.Snapshot, tfstates map[string]load.TerraformState) (results *Reconciled, err error) {
	// first load each item into memory, referenced by resourceType, resourceId, ARN
	var (
		configTypeIdMap = make(map[string]map[string]*load.ConfigurationItem)
		configArnMap    = make(map[string]*load.ConfigurationItem)
		// these are arns or type:id
		// eventually, we will handle this better
		itemToLocation = make(map[string]string)
	)
	for _, item := range snapshot.ConfigurationItems {
		if _, ok := configTypeIdMap[item.ResourceType]; !ok {
			configTypeIdMap[item.ResourceType] = make(map[string]*load.ConfigurationItem)
		}
		configTypeIdMap[item.ResourceType][item.ResourceID] = &item
		configArnMap[item.ARN] = &item
	}
	// now comes the harder part. We have to go through each tfstate and reconcile it with the snapshot
	// This would be easy if there were standards, but everything is driven by the provider,
	// terraform itself has no standard or intelligence about it, so we need to know all of them.
	for statefile, tfstate := range tfstates {
		for i, resource := range tfstate.Resources {
			if resource.Mode != load.TerraformManaged {
				continue
			}
			if resource.Provider != "provider.aws" && resource.Provider != `provider["registry.terraform.io/hashicorp/aws"]"` {
				continue
			}
			// look up the resource type
			var (
				configType string
				ok         bool
			)
			if configType, ok = awsTerraformToConfigTypeMap[resource.Type]; !ok {
				continue
			}
			for j, instance := range resource.Instances {
				var (
					resourceId, arn, location string
				)
				// try by arn first
				if arn, ok = instance.Attributes["id"].(string); ok {
					if _, ok := configArnMap[arn]; ok {
						location = "both"
					} else {
						location = "terraform"
					}
					itemToLocation[arn] = location
					continue
				}
				// there was no arn, so try resource type:ID
				if resourceId, ok = instance.Attributes["id"].(string); ok {
					if _, ok := configTypeIdMap[configType][resourceId]; ok {
						location = "both"
					} else {
						location = "terraform"
					}
					itemToLocation[fmt.Sprintf("%s:%s", configType, resourceId)] = location
					continue
				}
				// no resource ID or arn, which shouldn't occur, so warn
				log.Warnf("unable to find resource ID or ARN for resource %d, instance %d in file %s", i, j, statefile)
			}
		}
	}
	return &Reconciled{}, nil
}
