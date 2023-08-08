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
	resourceTypeEC2Instance           = "AWS::EC2::Instance"
	resourceTypeEksCluster            = "AWS::EKS::Cluster"
	resourceTypeASG                   = "AWS::AutoScaling::AutoScalingGroup"
	resourceTypeELB                   = "AWS::ElasticLoadBalancing::LoadBalancer"
	eksEniOwnerTagName                = "eks:eni:owner"
	eksEniOwnerTagValue               = "eks-vpc-resource-controller"
	terraformIAMPolicyType            = "aws_iam_policy"
	eksClusterOwnerTagNamePrefix      = "kubernetes.io/cluster/"
	owned                             = "owned"
	awsELBOwner                       = "amazon-elb"
	elbPrefix                         = "ELB "

	terraformAWSRegistryProvider = `provider["registry.terraform.io/hashicorp/aws"]"`
	terraformAWSProvider         = "provider.aws"
	terraformAWSProviderSuffix   = "provider.aws"
)
