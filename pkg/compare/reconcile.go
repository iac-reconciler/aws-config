package compare

import (
	"strings"

	"github.com/iac-reconciler/tf-aws-config/pkg/load"
	log "github.com/sirupsen/logrus"
)

type LocatedItem struct {
	*load.ConfigurationItem
	terraform  bool
	config     bool
	cfn        bool
	beanstalk  bool
	mappedType bool // indicates if the type was mapped between sources, or unique
}

// Reconcile reconcile the snapshot and tfstates.
func Reconcile(snapshot load.Snapshot, tfstates map[string]load.TerraformState) (items []*LocatedItem, err error) {
	// the keys are resource types, using the AWS-Config keys;
	// the values are map[string]*LocatedItem
	// in there, the keys are arn or id (if no arn), the values are
	// the LocatedItem, which includes marking where it was seen.
	var (
		itemToLocation = make(map[string]map[string]*LocatedItem)
		nameToLocation = make(map[string]map[string]*LocatedItem)
	)

	// we will do this in 2 passes. The first pass is to get the raw resources as they are
	// the second pass is to find those resources that contain other resources
	for _, item := range snapshot.ConfigurationItems {
		item := item // otherwise the pointer goes back to the original
		if item.ResourceType == configComplianceResourceType {
			continue
		}
		if item.ResourceType == "" {
			log.Warnf("AWS Config snapshot: empty resource type for item %s", item.ARN)
			continue
		}

		var mappedType = true
		if _, ok := awsConfigToTerraformTypeMap[item.ResourceType]; !ok {
			mappedType = false
		}
		if _, ok := itemToLocation[item.ResourceType]; !ok {
			itemToLocation[item.ResourceType] = make(map[string]*LocatedItem)
		}
		if _, ok := nameToLocation[item.ResourceType]; !ok {
			nameToLocation[item.ResourceType] = make(map[string]*LocatedItem)
		}
		key := item.ResourceID
		if key == "" {
			key = item.ARN
		}
		var (
			detail *LocatedItem
			ok     bool
		)
		if detail, ok = itemToLocation[item.ResourceType][key]; !ok {
			detail = &LocatedItem{
				ConfigurationItem: &item,
				mappedType:        mappedType,
			}
			itemToLocation[item.ResourceType][key] = detail
		}
		// we also map by name, if it exists, knowing it is a duplicate;
		// this is needed because the cloudformation and elasticbeanstalk stacks
		// sometimes reference a name, even though they call it an ID
		if _, ok := nameToLocation[item.ResourceType][item.ResourceName]; !ok {
			nameToLocation[item.ResourceType][item.ResourceName] = detail
		}
		detail.config = true

		// handle special resources that have children, e.g. routetable associations
		if item.ResourceType == resourceTypeRouteTable {
			// we will just create resources for these associations, as that is how AWSConfig
			// (sort of) sees it
			subType := resourceTypeRouteTableAssociation
			if _, ok := itemToLocation[subType]; !ok {
				itemToLocation[subType] = make(map[string]*LocatedItem)
			}
			if _, ok := nameToLocation[subType]; !ok {
				nameToLocation[subType] = make(map[string]*LocatedItem)
			}
			for _, assoc := range item.Configuration.Associations {
				itemToLocation[subType][assoc.AssociationID] = &LocatedItem{
					ConfigurationItem: &load.ConfigurationItem{
						ResourceType: subType,
						ResourceID:   assoc.AssociationID,
					},
					mappedType: true,
					config:     true,
				}
			}
		}
	}

	// second pass just for resources that contain others
	for _, item := range snapshot.ConfigurationItems {
		if item.ResourceType != resourceTypeStack && item.ResourceType != resourceTypeElasticBeanstalk {
			continue
		}

		// track subsidiary resources
		for _, resource := range item.Relationships {
			if resource.ResourceType == "" {
				log.Warnf("AWS Config snapshot: empty resource type for item %s", resource.ResourceID)
				continue
			}
			// only care about those contained
			if strings.TrimSpace(resource.Name) != resourceContains {
				continue
			}
			if _, ok := itemToLocation[resource.ResourceType]; !ok {
				itemToLocation[resource.ResourceType] = make(map[string]*LocatedItem)
			}
			var (
				detail *LocatedItem
				ok     bool
			)
			key := resource.ResourceID
			if key == "" {
				key = resource.ResourceName
			}
			if detail, ok = itemToLocation[resource.ResourceType][key]; !ok {
				// try by name
				if detail, ok = nameToLocation[resource.ResourceType][key]; !ok {
					log.Warnf("found unknown resource: %s %s", resource.ResourceType, key)
					continue
				}
			}
			switch item.ResourceType {
			case resourceTypeStack:
				detail.cfn = true
			case resourceTypeElasticBeanstalk:
				detail.beanstalk = true
			}
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
						ConfigurationItem: &load.ConfigurationItem{
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
