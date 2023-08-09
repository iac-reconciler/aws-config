package load

const (
	Regional = "Regional"
)

type Snapshot struct {
	FileVersion        string              `json:"fileVersion"`
	ConfigSnapShotID   string              `json:"configSnapshotId"`
	ConfigurationItems []ConfigurationItem `json:"configurationItems"`
}

type Relationship struct {
	ResourceType string `json:"resourceType"`
	ResourceID   string `json:"resourceId"`
	ResourceName string `json:"resourceName"`
	Name         string `json:"name"`
}

type ConfigurationItem struct {
	ResourceType  string            `json:"resourceType"`
	ResourceID    string            `json:"resourceId"`
	ResourceName  string            `json:"resourceName"`
	ARN           string            `json:"ARN"`
	Region        string            `json:"awsRegion"` // should be limited to certain regions
	Zone          string            `json:"availabilityZone"`
	AccountID     string            `json:"awsAccountId"`            // should be limited to numeric
	Status        string            `json:"configurationItemStatus"` // should be limited to the limited sets of status
	Relationships []Relationship    `json:"relationships"`
	Configuration Configuration     `json:"configuration"`
	Tags          map[string]string `json:"tags"`
}

type Configuration struct {
	Associations        []Association  `json:"associations"`
	Association         Association    `json:"association"`
	NetworkInterfaceIDs []string       `json:"networkInterfaceIds"`
	Instances           []EC2Instance  `json:"instances,omitempty"`
	Description         string         `json:"description"`
	InterfaceType       string         `json:"interfaceType,omitempty"`
	Attachment          Attachment     `json:"attachment,omitempty"`
	Namespace           string         `json:"namespace,omitempty"`
	Dimensions          []Dimension    `json:"dimensions,omitempty"`
	IPPermissions       []IPPermission `json:"ipPermissions,omitempty"`
	IPPermissionsEgress []IPPermission `json:"ipPermissionsEgress,omitempty"`
	Routes              []Route        `json:"routes,omitempty"`
}

type Association struct {
	AssociationID string `json:"routeTableAssociationId,omitempty"`
	RouteTableID  string `json:"routeTableId,omitempty"`
	SubnetID      string `json:"subnetId,omitempty"`
	IPOwnerID     string `json:"ipOwnerId,omitempty"`
	PublicDNSName string `json:"publicDnsName,omitempty"`
	PublicIP      string `json:"publicIp,omitempty"`
}

type EC2Instance struct {
	InstanceID       string `json:"instanceId"`
	InstanceType     string `json:"instanceType"`
	AvailabilityZone string `json:"availabilityZone"`
}

type Attachment struct {
	AttachmentID    string `json:"attachmentId"`
	InstanceOwnerID string `json:"instanceOwnerId"`
}

type Dimension struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type IPPermission struct {
	FromPort         int64             `json:"fromPort"`
	ToPort           int64             `json:"toPort"`
	IPProtocol       string            `json:"ipProtocol"`
	IPRanges         []string          `json:"ipRanges"`
	IPV4Ranges       []IPV4Range       `json:"ipv4Ranges"`
	UserIDGroupPairs []UserIDGroupPair `json:"userIdGroupPairs"`
}

type IPV4Range struct {
	CIDRIP string `json:"cidrIp"`
}

type UserIDGroupPair struct {
	Description          string `json:"description"`
	GroupID              string `json:"groupId"` // this is the source security group
	UserID               string `json:"userId"`
	VPCID                string `json:"vpcId"`
	VPCPeeringConnection string `json:"vpcPeeringConnection"`
}

type Route struct {
	DestinationCIDRBlock   string `json:"destinationCidrBlock,omitempty"`
	Origin                 string `json:"origin,omitempty"`
	State                  string `json:"state,omitempty"`
	VPCPeeringConnectionID string `json:"vpcPeeringConnectionId,omitempty"`
	GatewayID              string `json:"gatewayId,omitempty"`
	NATGatewayID           string `json:"natGatewayId,omitempty"`
}
