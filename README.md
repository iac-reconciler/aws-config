# AWS Config to Infrastructure-as-Code Reconciler

Utility to enable reconciliation between AWS Config snapshot, and one or more Terraform
statefiles, as well as possibly other IaC tools in the future.

This does not (yet) contact AWS and do a report for you. It may at some point.
For now, you need to enable [AWS Config](https://aws.amazon.com/config/), save
the snapshot output to a bucket, download it, and provide it as an input
file to this utility.

## Usage

```bash
$ aws-config path/to/aws-config-snapshot.json path/to/terraform.tfstate
```

As many organizations split Terraform into multiple configs, each with their own
statefile, you can tell it to search in a path and find all `*.tfstate` files:

```bash
$ aws-config generate path/to/aws-config-snapshot.json path/to/terraform/root --tf-recursive
```

## Output

The output is a list of all resources found in one, the other, or both.
Each resource lists which it is found within, and the resource's type and ID.

You can get summary statistics using the `--summary`.

## Limitations

As of this writing, everything is stored in memory. This should not be an issue except
at very large scale. We are open to replacing it with a memory-mapped file, or an embedded
sql database like sqlite, if this becomes an issue.
