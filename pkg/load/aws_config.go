package load

const (
	Regional = "Regional"
)

type Snapshot struct {
	FileVersion        string              `json:"fileVersion"`
	ConfigSnapShotID   string              `json:"configSnapshotId"`
	ConfigurationItems []ConfigurationItem `json:"configurationItems"`
}

type ConfigurationItem struct {
	ResourceType string `json:"resourceType"`
	ResourceID   string `json:"resourceId"`
	ARN          string `json:"ARN"`
	Region       string `json:"awsRegion"` // should be limited to certain regions
	Zone         string `json:"availabilityZone"`
	AccountID    string `json:"awsAccountId"`            // should be limited to numeric
	Status       string `json:"configurationItemStatus"` // should be limited to the limited sets of status
}
