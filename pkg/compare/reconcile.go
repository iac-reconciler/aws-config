package compare

import (
	"github.com/iac-reconciler/tf-aws-config/pkg/load"
	log "github.com/sirupsen/logrus"
)

type LocatedItem struct {
	load.ConfigurationItem
	terraform  bool
	config     bool
	mappedType bool // indicates if the type was mapped between sources, or unique
}

// Reconcile reconcile the snapshot and tfstates.
// Not yet implemented, so returns an empty struct
func Reconcile(snapshot load.Snapshot, tfstates map[string]load.TerraformState) (items []*LocatedItem, err error) {
	// the keys are resource types, using the AWS-Config keys;
	// the values are map[string]*LocatedItem
	// in there, the keys are arn or id (if no arn), the values are
	// the LocatedItem, which includes marking where it was seen.
	var itemToLocation = make(map[string]map[string]*LocatedItem)

	for _, item := range snapshot.ConfigurationItems {
		if item.ResourceType == configComplianceResourceType {
			continue
		}
		if item.ResourceType == "" {
			log.Warnf("AWS Config snapshot: empty resource type for item %s", item.ARN)
			continue
		}
		var mappedType = true
		if _, ok := awsConfigToTerraformypeMap[item.ResourceType]; !ok {
			log.Warnf("unknown AWS Config resource type: %s", item.ResourceType)
			mappedType = false
		}
		if _, ok := itemToLocation[item.ResourceType]; !ok {
			itemToLocation[item.ResourceType] = make(map[string]*LocatedItem)
		}
		key := item.ARN
		if key == "" {
			key = item.ResourceID
		}
		itemToLocation[item.ResourceType][key] = &LocatedItem{
			ConfigurationItem: item,
			config:            true,
			mappedType:        mappedType,
		}
	}
	// now comes the harder part. We have to go through each tfstate and reconcile it with the snapshot
	// This would be easy if there were standards, but everything is driven by the provider,
	// terraform itself has no standard or intelligence about it, so we need to know all of them.
	for statefile, tfstate := range tfstates {
		for i, resource := range tfstate.Resources {
			// only care about managed resources
			if resource.Mode != load.TerraformManaged {
				continue
			}
			// only care about aws resources
			if resource.Provider != "provider.aws" && resource.Provider != `provider["registry.terraform.io/hashicorp/aws"]"` {
				continue
			}
			// look up the resource type
			var (
				configType string
				ok         bool
				mappedType = true
			)
			if configType, ok = awsTerraformToConfigTypeMap[resource.Type]; !ok {
				log.Warnf("unknown terraform resource type: %s", resource.Type)
				configType = resource.Type
				mappedType = false
			}
			if _, ok := itemToLocation[configType]; !ok {
				itemToLocation[configType] = make(map[string]*LocatedItem)
			}
			for j, instance := range resource.Instances {
				var (
					resourceId, arn string
					item            *LocatedItem
					key             string
				)
				// try by arn first
				arnPtr := instance.Attributes["arn"]
				if arnPtr != nil {
					arn = arnPtr.(string)
				}
				resourceIdPtr := instance.Attributes["id"]
				if resourceIdPtr != nil {
					resourceId = resourceIdPtr.(string)
				}
				if arn != "" {
					key = arn
				} else if resourceId != "" {
					key = resourceId
				} else {
					log.Warnf("unable to find resource ID or ARN for resource %d, instance %d in file %s", i, j, statefile)
					continue
				}
				item = itemToLocation[configType][key]

				// if we could not find it by ARN or by configType+id, then
				// it is only in terraform
				if item == nil {
					item = &LocatedItem{
						ConfigurationItem: load.ConfigurationItem{
							ResourceType: configType,
							ResourceID:   resourceId,
							ARN:          arn,
						},
						mappedType: mappedType,
					}
					itemToLocation[configType][key] = item
				}
				item.terraform = true
			}
		}
	}

	// now we have all of the resources listed as in Terraform, Config or both
	// so create the reconciled info
	for _, locations := range itemToLocation {
		for _, item := range locations {
			items = append(items, item)
		}
	}
	return items, nil
}
