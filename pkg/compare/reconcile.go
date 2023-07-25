package compare

import (
	"github.com/iac-reconciler/tf-aws-config/pkg/load"
	log "github.com/sirupsen/logrus"
)

type Reconciled struct {
	TerraformOnly []load.ConfigurationItem `json:"terraform_only"`
	ConfigOnly    []load.ConfigurationItem `json:"config_only"`
	Both          []load.ConfigurationItem `json:"both"`
}

// Reconcile reconcile the snapshot and tfstates.
// Not yet implemented, so returns an empty struct
func Reconcile(snapshot load.Snapshot, tfstates map[string]load.TerraformState) (results *Reconciled, err error) {
	// first load each item into memory, referenced by resourceType, resourceId, ARN
	var (
		configTypeIdMap = make(map[string]map[string]*load.ConfigurationItem)
		configArnMap    = make(map[string]*load.ConfigurationItem)
		// the keys are resource types, using the AWS-Config keys;
		// the values are map[string]string
		// in there, the keys are arn or id (if no arn), the values are location,
		// one of "terraform", "config", "both"
		itemToLocation = make(map[string]map[string]string)
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
			if _, ok := itemToLocation[configType]; !ok {
				itemToLocation[configType] = make(map[string]string)
			}
			if _, ok := configTypeIdMap[configType]; !ok {
				configTypeIdMap[configType] = make(map[string]*load.ConfigurationItem)
			}
			for j, instance := range resource.Instances {
				var (
					resourceId, arn, location string
					item                      *load.ConfigurationItem
					ok                        bool
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
					if item, ok = configArnMap[arn]; ok {
						location = "both"
					} else {
						location = "terraform"
					}
				}
				if resourceId != "" {
					if item, ok = configTypeIdMap[configType][resourceId]; ok {
						location = "both"
					} else {
						location = "terraform"
					}
				}
				if item == nil {
					item = &load.ConfigurationItem{
						ResourceType: configType,
						ResourceID:   resourceId,
						ARN:          arn,
					}
				}
				// if it only is in Terraform, then we do not have it in the maps
				// so we need to add it for later reference
				if location == "terraform" {
					if item.ARN != "" {
						configArnMap[item.ARN] = item
					}
					if item.ResourceID != "" {
						configTypeIdMap[configType][item.ResourceID] = item
					}
				}
				if location != "" {
					key := arn
					if key == "" {
						key = resourceId
					}
					itemToLocation[configType][key] = location
					continue
				}
				// no resource ID or arn, which shouldn't occur, so warn
				log.Warnf("unable to find resource ID or ARN for resource %d, instance %d in file %s", i, j, statefile)
			}
		}
	}
	// go through config and see what is not covered already in terraform
	for _, item := range snapshot.ConfigurationItems {
		// we do not care about the "rules reporting resources" from AWS Config
		if item.ResourceType == configComplianceResourceType {
			continue
		}
		if _, ok := itemToLocation[item.ResourceType]; !ok {
			itemToLocation[item.ResourceType] = make(map[string]string)
		}
		key := item.ARN
		if key == "" {
			key = item.ResourceID
		}
		if _, ok := itemToLocation[item.ResourceType][key]; !ok {
			itemToLocation[item.ResourceType][key] = "config"
		}
	}
	// now we have all of the resources listed as in Terraform, Config or both
	// so create the reconciled info
	results = &Reconciled{}
	for resourceType, locations := range itemToLocation {
		for key, location := range locations {
			var configItem *load.ConfigurationItem
			if item, ok := configArnMap[key]; ok {
				configItem = item
			} else if item, ok := configTypeIdMap[resourceType][key]; ok {
				configItem = item
			} else {
				log.Errorf("could not find item %s in config maps", key)
				continue
			}
			switch location {
			case "terraform":
				results.TerraformOnly = append(results.TerraformOnly, *configItem)
			case "config":
				results.ConfigOnly = append(results.ConfigOnly, *configItem)
			case "both":
				results.Both = append(results.Both, *configItem)
			}
		}
	}
	return results, nil
}
