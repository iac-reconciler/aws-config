package compare

import (
	"github.com/iac-reconciler/tf-aws-config/pkg/load"
)

type Reconciled struct {
}

// Reconcile reconcile the snapshot and tfstates.
// Not yet implemented, so returns an empty struct
func Reconcile(snapshot load.Snapshot, tfstates []load.TerraformState) (results *Reconciled, err error) {
	return &Reconciled{}, nil
}
