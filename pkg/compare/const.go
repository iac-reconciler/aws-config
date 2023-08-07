package compare

const (
	configComplianceResourceType      = "AWS::Config::ResourceCompliance"
	resourceContains                  = "Contains"
	resourceAttachedToInstance        = "Is attached to Instance"
	resourceTypeStack                 = "AWS::CloudFormation::Stack"
	resourceTypeElasticBeanstalk      = "AWS::ElasticBeanstalk::Application"
	resourceTypeRouteTable            = "AWS::EC2::RouteTable"
	resourceTypeRouteTableAssociation = "AWS::EC2::SubnetRouteTableAssociation"
	resourceTypeVPCEndpoint           = "AWS::EC2::VPCEndpoint"
	resourceTypeENI                   = "AWS::EC2::NetworkInterface"
	resourceTypeEBSVolume             = "AWS::EC2::Volume"
	resourceTypeEksCluster            = "AWS::EKS::Cluster"
	resourceTypeTerraform             = "Terraform"
	eksEniOwnerTagName                = "eks:eni:owner"
	eksEniOwnerTagValue               = "eks-vpc-resource-controller"
	eksClusterOwnerTagNamePrefix      = "kubernetes.io/cluster/"
	owned                             = "owned"
)
