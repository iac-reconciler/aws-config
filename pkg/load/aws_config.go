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
	NetworkInterfaceIDs []string      `json:"networkInterfaceIds"`
	Instances           []EC2Instance `json:"instances,omitempty"`
}

type Association struct {
	AssociationID string `json:"routeTableAssociationId"`
	RouteTableID  string `json:"routeTableId"`
	SubnetID      string `json:"subnetId"`
}

type EC2Instance struct {
	InstanceID       string `json:"instanceId"`
	InstanceType     string `json:"instanceType"`
	AvailabilityZone string `json:"availabilityZone"`
}
