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
	Associations        []Association `json:"associations"`
	Association         Association   `json:"association"`
	NetworkInterfaceIDs []string      `json:"networkInterfaceIds"`
	Instances           []EC2Instance `json:"instances,omitempty"`
	Description         string        `json:"description"`
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
