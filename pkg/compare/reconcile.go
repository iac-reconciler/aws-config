package compare

import (
	"strings"

	"github.com/iac-reconciler/aws-config/pkg/load"
	log "github.com/sirupsen/logrus"
)

// LocatedItem is a configuration item that has been located in a source.
// That source could be a snapshot, a terraform state, or both.
// It also includes a parent, if any.
type LocatedItem struct {
	*load.ConfigurationItem
	config     bool
	terraform  bool
	parent     *LocatedItem
	mappedType bool // indicates if the type was mapped between sources, or unique
}

func (l LocatedItem) Source(src string) bool {
	switch strings.ToLower(src) {
	case "config":
		return l.config
	case "terraform":
		return l.terraform
	case "owned":
		return l.Owned()
	default:
		return false
	}
}

func (l LocatedItem) Owned() bool {
	return l.terraform || l.parent != nil
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

		// handle special resources that have children

		// routetable associations
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
		// get the correct LocatedItem pointer for this item
		var (
			located *LocatedItem
			ok      bool
		)
		key := item.ResourceID
		if key == "" {
			key = item.ARN
		}
		if located, ok = itemToLocation[item.ResourceType][key]; !ok {
			log.Warnf("found unknown resource: %s %s", item.ResourceType, key)
			continue
		}

		// EKS-created ENIs
		if item.ResourceType == resourceTypeENI {
			var (
				clusterName string
				eniTag      bool
			)
			for tagName, tagValue := range item.Tags {
				if strings.HasPrefix(tagName, eksClusterOwnerTagNamePrefix) && tagValue == owned {
					clusterName = strings.TrimPrefix(tagName, eksClusterOwnerTagNamePrefix)
				}
				if tagName == eksEniOwnerTagName && tagValue == eksEniOwnerTagValue {
					eniTag = true
				}
			}
			if !eniTag {
				continue
			}
			if clusterName != "" {
				// this is an EKS-created ENI
				// find the parent, and mark it
				if eks, ok := itemToLocation[resourceTypeEksCluster]; ok {
					if parent, ok := eks[clusterName]; ok {
						located.parent = parent
					}
				}
			}
		}

		if item.ResourceType == resourceTypeStack || item.ResourceType == resourceTypeElasticBeanstalk {
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
				detail.parent = located
			}
		}

		// VPC-Endpoint-owned ENIs
		if item.ResourceType == resourceTypeVPCEndpoint {
			// we will just create resources for these associations, as that is how AWSConfig
			// sees it
			subType := resourceTypeENI
			if _, ok := itemToLocation[subType]; !ok {
				itemToLocation[subType] = make(map[string]*LocatedItem)
			}
			if _, ok := nameToLocation[subType]; !ok {
				nameToLocation[subType] = make(map[string]*LocatedItem)
			}
			for _, eni := range item.Configuration.NetworkInterfaceIDs {
				var (
					detail *LocatedItem
					ok     bool
				)
				if detail, ok = itemToLocation[subType][eni]; !ok {
					log.Warnf("found unknown resource: %s %s", subType, eni)
					continue
				}
				detail.parent = located
			}
		}

		// EC2-instance owned volumes
		if item.ResourceType == resourceTypeEBSVolume {
			// indicate that it is owned by whatever it is attached to
			for _, resource := range item.Relationships {
				if resource.ResourceType == "" {
					log.Warnf("AWS Config snapshot: empty resource type for item %s", resource.ResourceID)
					continue
				}
				// only care about those attached-to
				if strings.TrimSpace(resource.Name) != resourceAttachedToInstance {
					continue
				}
				if _, ok := itemToLocation[resource.ResourceType]; !ok {
					itemToLocation[resource.ResourceType] = make(map[string]*LocatedItem)
				}
				key := resource.ResourceID
				if key == "" {
					key = resource.ResourceName
				}
				var (
					detail *LocatedItem
				)
				if detail, ok = itemToLocation[resource.ResourceType][key]; !ok {
					// try by name
					if detail, ok = nameToLocation[resource.ResourceType][key]; !ok {
						log.Warnf("found unknown resource: %s %s", resource.ResourceType, key)
						continue
					}
				}
				located.parent = detail
			}
		}

		// ASG-owned ec2 instances
		if item.ResourceType == resourceTypeASG {
			// indicate that it is owned by whatever it is attached to
			for _, instance := range item.Configuration.Instances {
				if _, ok := itemToLocation[resourceTypeEC2Instance]; !ok {
					itemToLocation[resourceTypeEC2Instance] = make(map[string]*LocatedItem)
				}
				var (
					detail *LocatedItem
				)
				if detail, ok = itemToLocation[resourceTypeEC2Instance][instance.InstanceID]; !ok {
					log.Warnf("found unknown resource: %s %s", resourceTypeEC2Instance, key)
					continue
				}
				detail.parent = located
			}
		}

		// ENIs that are owned by an ELB
		if item.ResourceType == resourceTypeENI {
			if item.Configuration.Association.IPOwnerID != awsELBOwner {
				continue
			}
			// find the ELB that owns it, make it the parent
			if !strings.HasPrefix(item.Configuration.Description, elbPrefix) {
				continue
			}
			elbName := strings.TrimPrefix(item.Configuration.Description, elbPrefix)
			// now find the correct ELB
			var elbMap map[string]*LocatedItem
			if elbMap, ok = itemToLocation[resourceTypeELB]; !ok {
				continue
			}
			if elb, ok := elbMap[elbName]; !ok {
				continue
			} else {
				located.parent = elb
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
			if resource.Provider != terraformAWSProvider &&
				resource.Provider != terraformAWSRegistryProvider &&
				!strings.HasSuffix(resource.Provider, terraformAWSProviderSuffix) {
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
					resourceId, arn, name string
					item                  *LocatedItem
					key                   string
				)
				// try by arn first - some, however, prioritize others. We need the one that matches the resourceId
				arnPtr := instance.Attributes["arn"]
				if arnPtr != nil {
					arn = arnPtr.(string)
				}
				resourceIdPtr := instance.Attributes["id"]
				if resourceIdPtr != nil {
					resourceId = resourceIdPtr.(string)
				}
				namePtr := instance.Attributes["name"]
				if namePtr != nil {
					name = namePtr.(string)
				}

				switch {
				case arn != "":
					key = arn
				case resourceId != "":
					key = resourceId
				default:
					log.Warnf("unable to find resource ID or ARN for resource %d, instance %d in file %s", i, j, statefile)
					continue
				}

				item, ok = itemToLocation[configType][key]
				if !ok {
					item, ok = nameToLocation[configType][name]
					if !ok {
						// if we could not find it by ARN or by configType+id or configType+name, then
						// it is only in terraform
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
				}

				item.terraform = true
			}
		}
	}

	for _, locations := range itemToLocation {
		for _, item := range locations {
			items = append(items, item)
		}
	}
	return items, nil
}
