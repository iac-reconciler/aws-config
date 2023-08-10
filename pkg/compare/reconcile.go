package compare

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
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

func (l LocatedItem) Ephemeral() bool {
	return !l.terraform && !l.config
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
		arnToLocation  = make(map[string]map[string]*LocatedItem)

		uuidSize = len(uuid.New().String())
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
		if _, ok := arnToLocation[item.ResourceType]; !ok {
			arnToLocation[item.ResourceType] = make(map[string]*LocatedItem)
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
		if item.ARN != "" {
			arnToLocation[item.ResourceType][item.ARN] = detail
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

	// second pass for CloudFormation-owned resources
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

		// CloudFormation and Beanstalk created items
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
						detail = &LocatedItem{
							ConfigurationItem: &load.ConfigurationItem{
								ResourceType: resource.ResourceType,
								ResourceID:   resource.ResourceID,
							},
						}
						itemToLocation[resource.ResourceType][resource.ResourceID] = detail
					}
				}
				detail.parent = located
			}

			for _, resource := range item.SupplementaryConfiguration.UnsupportedResources {
				if resource.ResourceType == "" {
					log.Warnf("AWS Config snapshot: empty resource type for item %s", resource.ResourceID)
					continue
				}
				if resource.ResourceID == "" {
					log.Warnf("AWS Config snapshot: empty resource ID for item %s", resource.ResourceType)
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
				if detail, ok = itemToLocation[resource.ResourceType][key]; !ok {
					// try by name
					if detail, ok = nameToLocation[resource.ResourceType][key]; !ok {
						detail = &LocatedItem{
							ConfigurationItem: &load.ConfigurationItem{
								ResourceType: resource.ResourceType,
								ResourceID:   resource.ResourceID,
							},
						}
						itemToLocation[resource.ResourceType][resource.ResourceID] = detail
					}
				}
				detail.parent = located
			}
		}
	}

	// third pass just for resources that contain others
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
			// handle RDS instance-owned ENIs; which, unfortunately, are not tagged on either side
			// who would believe it?
			if item.Configuration.Description == rdsENI {
				located.parent = &LocatedItem{
					ConfigurationItem: &load.ConfigurationItem{
						ResourceType: resourceTypeRDSInstance,
					},
				}
			}

			var (
				clusterName string
				eniTag      bool
				nodeId      string
			)
			for tagName, tagValue := range item.Tags {
				if strings.HasPrefix(tagName, eksClusterOwnerTagNamePrefix) && tagValue == owned {
					clusterName = strings.TrimPrefix(tagName, eksClusterOwnerTagNamePrefix)
				}
				if tagName == eksEniOwnerTagName && tagValue == eksEniOwnerTagValue {
					eniTag = true
				}
				if tagName == k8sInstanceTag {
					nodeId = tagValue
				}
			}
			switch {
			case eniTag && clusterName != "":
				// this is an EKS-created ENI
				// find the parent, and mark it
				if resources, ok := itemToLocation[resourceTypeEksCluster]; ok {
					if parent, ok := resources[clusterName]; ok {
						located.parent = parent
					}
				}
			case nodeId != "":
				// this is a EC2 instance-created ENI
				// find the parent, and mark it
				if resources, ok := itemToLocation[resourceTypeEC2Instance]; ok {
					if parent, ok := resources[nodeId]; ok {
						located.parent = parent
					}
				}
			}
		}

		// EC2-created instances
		if item.ResourceType == resourceTypeEC2Instance {
			for _, rel := range item.Relationships {
				if rel.ResourceType == resourceTypeENI {
					// find the parent, and mark it
					if resources, ok := itemToLocation[resourceTypeENI]; ok {
						if eni, ok := resources[rel.ResourceID]; ok {
							eni.parent = located
						}
					}
				}
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

		// cloudwatch alarms
		if item.ResourceType == resourceTypeAlarm {
			switch item.Configuration.Namespace {
			case cloudWatchNamespaceELB:
				// find the ELB
				var lbID string
				for _, dim := range item.Configuration.Dimensions {
					if dim.Name != dimensionLoadBalancerName {
						continue
					}
					lbID = dim.Value
					break
				}
				if elbMap, ok := itemToLocation[resourceTypeELB]; ok {
					if elb, ok := elbMap[lbID]; ok {
						located.parent = elb
					}

				}
			}
		}

		// EC2Fleets can be owned by a LaunchTemplate
		if item.ResourceType == resourceTypeEC2Fleet {
			for _, ltConfig := range item.Configuration.LaunchTemplateConfigs {
				if ltConfig.LaunchTemplateSpecification.LaunchTemplateID != "" {
					if _, ok := itemToLocation[resourceTypeLaunchTemplate]; ok {
						if lt, ok := itemToLocation[resourceTypeLaunchTemplate][ltConfig.LaunchTemplateSpecification.LaunchTemplateID]; ok {
							located.parent = lt
						}
					}
				}
			}
		}

		// EKS-created SecurityGroups
		if item.ResourceType == resourceTypeSecurityGroup {
			var (
				clusterName string
			)
			for tagName, tagValue := range item.Tags {
				if strings.HasPrefix(tagName, eksClusterOwnerTagNamePrefix) && tagValue == owned {
					clusterName = strings.TrimPrefix(tagName, eksClusterOwnerTagNamePrefix)
				}
				break
			}
			if clusterName != "" {
				// this is an EKS-created ENI
				// find the parent, and mark it
				if resources, ok := itemToLocation[resourceTypeEksCluster]; ok {
					if parent, ok := resources[clusterName]; ok {
						located.parent = parent
					}
				}
			}
		}

		// ENIs that are owned by an ELB
		if item.ResourceType == resourceTypeENI {
			switch {
			case item.Configuration.InterfaceType == lambdaInterfaceType && strings.HasPrefix(item.Configuration.Description, lambdaPrefix):
				lambdaName := strings.TrimPrefix(item.Configuration.Description, lambdaPrefix)
				// lambda also includes a UUID at the end, so we need to remove that
				if len(lambdaName) > uuidSize {
					// check if it finishes with a UUID
					if _, err := uuid.Parse(lambdaName[len(lambdaName)-uuidSize:]); err == nil {
						lambdaName = lambdaName[:len(lambdaName)-uuidSize]
						// remove last -
						if lambdaName[len(lambdaName)-1] == '-' {
							lambdaName = lambdaName[:len(lambdaName)-1]
						}
					}
				}
				// now find the correct lambda
				if lambdaMap, ok := itemToLocation[resourceTypeLambda]; ok {
					if lambda, ok := lambdaMap[lambdaName]; ok {
						located.parent = lambda
					}
				}

			case strings.HasPrefix(item.Configuration.Description, elbPrefix) &&
				(item.Configuration.Association.IPOwnerID == awsELBOwner ||
					item.Configuration.Attachment.InstanceOwnerID == awsELBOwner ||
					item.Configuration.InterfaceType == nlb):
				// find the ELB that owns it, make it the parent
				region := item.Region
				account := item.AccountID
				elbName := strings.TrimPrefix(item.Configuration.Description, elbPrefix)
				// could be ELB or ELBv2; nothing in it indicates that it is, except perhaps the start of the name,
				// so we might as well just check both
				var found bool
				if elbMap, ok := itemToLocation[resourceTypeELB]; ok {
					if elb, ok := elbMap[elbName]; ok {
						located.parent = elb
						found = true
					}
				}
				if !found {
					if region != "" && account != "" {
						nlbArn := fmt.Sprintf("%s:%s:%s:loadbalancer/%s", elbArnPrefix, region, account, elbName)
						if elbMap, ok := itemToLocation[resourceTypeELBV2]; ok {
							if elb, ok := elbMap[nlbArn]; ok {
								located.parent = elb
							}
						}
					}
				}
			case item.Configuration.InterfaceType == natGatewayInterfaceType && strings.HasPrefix(item.Configuration.Description, natGatewayPrefix):
				itemName := strings.TrimPrefix(item.Configuration.Description, natGatewayPrefix)
				// now find the correct NAT Gateway
				if itemMap, ok := itemToLocation[resourceTypeNATGateway]; ok {
					if item, ok := itemMap[itemName]; ok {
						located.parent = item
					}
				}
			case strings.HasPrefix(item.Configuration.Description, elastiCachePrefix):
				itemName := strings.TrimPrefix(item.Configuration.Description, elastiCachePrefix)
				// now find the correct ElastiCache Cluster
				if itemMap, ok := itemToLocation[resourceTypeElastiCacheCluster]; ok {
					if item, ok := itemMap[itemName]; ok {
						located.parent = item
					}
				}
			case item.Configuration.InterfaceType == transitGatewayInterfaceType && strings.HasPrefix(item.Configuration.Description, transitGatewayPrefix):
				itemName := strings.TrimPrefix(item.Configuration.Description, transitGatewayPrefix)
				// now find the correct ElastiCache Cluster
				if itemMap, ok := itemToLocation[resourceTypeTransitGatewayAttachment]; ok {
					if item, ok := itemMap[itemName]; ok {
						located.parent = item
					}
				}
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
				!strings.HasSuffix(resource.Provider, terraformAWSProvider) &&
				!strings.HasSuffix(resource.Provider, terraformAWSRegistryProvider) {
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
					parentFound           bool
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

				// some types have special rules
				switch configType {
				case terraformTypeSecurityGroupRule:
					// check if the security group exists
					var (
						securityGroupID string
					)
					securityGroupIDPtr := instance.Attributes["security_group_id"]
					if securityGroupIDPtr != nil {
						securityGroupID = securityGroupIDPtr.(string)
					}

					// find the security group in Config based on the ID
					if securityGroupID != "" {
						// if we could not find the security group, then nothing to look for in Config; it only is in terraform
						if item, ok = itemToLocation[resourceTypeSecurityGroup][securityGroupID]; !ok {
							if item, ok = nameToLocation[resourceTypeSecurityGroup][securityGroupID]; !ok {
								item = nil
							}
						}
					}
					// we found the parent security group, look through the rules and find the one that matches
					if item != nil {
						typePtr := instance.Attributes["type"]
						var (
							ruleType string
						)
						if typePtr != nil {
							ruleType = typePtr.(string)
						}
						switch ruleType {
						case ingress:
							// find the rule in the security group
							for _, rule := range item.Configuration.IPPermissions {
								if rule.FromPort != instance.Attributes["from_port"] ||
									rule.ToPort != instance.Attributes["to_port"] ||
									rule.IPProtocol != instance.Attributes["protocol"] {
									continue
								}
								for _, pair := range rule.UserIDGroupPairs {
									if pair.GroupID != instance.Attributes["source_security_group_id"] ||
										pair.Description != instance.Attributes["description"] {
										continue
									}
									// we have a match
									parentFound = true
									break
								}
								if parentFound {
									break
								}
							}
						case egress:
							// find the rule in the security group
							for _, rule := range item.Configuration.IPPermissionsEgress {
								if rule.FromPort != instance.Attributes["from_port"] ||
									rule.ToPort != instance.Attributes["to_port"] ||
									rule.IPProtocol != instance.Attributes["protocol"] {
									continue
								}
								for _, pair := range rule.UserIDGroupPairs {
									if pair.GroupID != instance.Attributes["source_security_group_id"] ||
										pair.Description != instance.Attributes["description"] {
										continue
									}
									// we have a match
									parentFound = true
									break
								}
								if parentFound {
									break
								}
							}

						default:
							// unknown rule type, so just skip it
							log.Warnf("unknown security group rule type %s for resource %d, instance %d in file %s", ruleType, i, j, statefile)
						}
					}
				case terraformTypeRoute, resourceTypeRoute:
					// check if the route table exists
					var (
						routeTable string
					)
					routeTablePtr := instance.Attributes["route_table_id"]
					if routeTablePtr != nil {
						routeTable = routeTablePtr.(string)
					}

					// find the route table in Config based on the ID
					if resourceTypeRouteTable != "" {
						// if we could not find the route table, then nothing to look for in Config; it only is in terraform
						if item, ok = itemToLocation[resourceTypeRouteTable][routeTable]; !ok {
							if item, ok = nameToLocation[resourceTypeRouteTable][routeTable]; !ok {
								item = nil
							}
						}
					}
					// we found the parent route table, look through the routes and find the one that matches
					if item != nil {
						for _, route := range item.Configuration.Routes {
							if route.DestinationCIDRBlock == instance.Attributes["destination_cidr_block"] &&
								route.Origin == instance.Attributes["origin"] &&
								route.VPCPeeringConnectionID == instance.Attributes["vpc_peering_connection_id"] &&
								route.GatewayID == instance.Attributes["gateway_id"] &&
								route.NATGatewayID == instance.Attributes["nat_gateway_id"] {
								parentFound = true
								break
							}
						}
					}
				case terraformTypeRolePolicyAttachment:
					// check if the role exists
					var (
						roleID       string
						policyID     string
						role, policy *LocatedItem
					)
					rolePtr := instance.Attributes["role"]
					if rolePtr != nil {
						roleID = rolePtr.(string)
					}
					policyPtr := instance.Attributes["policy_arn"]
					if policyPtr != nil {
						policyID = policyPtr.(string)
					}
					if roleID != "" {
						if role, ok = itemToLocation[resourceTypeIAMRole][roleID]; !ok {
							if role, ok = nameToLocation[resourceTypeIAMRole][roleID]; !ok {
								role = nil
							}
						}
					}
					if policyID != "" {
						if policy, ok = arnToLocation[resourceTypeIAMPolicy][policyID]; !ok {
							policy = nil
						}
					}
					if role != nil && policy != nil {
						for _, rel := range policy.Relationships {
							if rel.ResourceType == resourceTypeIAMRole &&
								rel.Name == nameRoleAttached &&
								(rel.ResourceID == roleID || rel.ResourceID == role.ConfigurationItem.ResourceID) {
								parentFound = true
								break
							}
						}
					}
				default:
					if item, ok = itemToLocation[configType][key]; !ok {
						if item, ok = nameToLocation[configType][name]; !ok {
							if item, ok = arnToLocation[configType][arn]; !ok {
								item = nil
							}
						}
					}
				}
				// if we could not find the security group, then nothing to look for in Config; it only is in terraform
				// It is found if we found the item, or if we found a parent.
				if item == nil && !parentFound {
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
				if item != nil {
					item.terraform = true
				}
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
