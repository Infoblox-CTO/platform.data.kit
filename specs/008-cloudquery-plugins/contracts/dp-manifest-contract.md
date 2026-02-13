# CloudQuery dp.yaml Manifest Contract

## Source Plugin (Python, default)

```yaml
apiVersion: cdpp.io/v1alpha1
kind: DataPackage
metadata:
  name: "{{.Name}}"
  namespace: "{{.Namespace}}"
spec:
  type: cloudquery
  description: "CloudQuery source plugin for {{.Name}}"
  owner: "{{.Owner}}"
  cloudquery:
    role: source
    tables:
      - example_resource
    grpcPort: 7777
    concurrency: 10000
  runtime:
    image: "{{.Namespace}}/{{.Name}}:latest"
```

## Source Plugin (Go)

Identical manifest structure — only the scaffolded source files differ.

## Destination Plugin (reserved, not yet supported)

```yaml
# NOT scaffolded — dp init returns "not yet supported"
spec:
  type: cloudquery
  cloudquery:
    role: destination
```

## Validation Rules

| Field | Rule | Error Code |
|-------|------|------------|
| `spec.cloudquery` | Required when `type=cloudquery` | E060 |
| `spec.cloudquery.role` | Required, must be `source` or `destination` | E061 |
| `spec.cloudquery.role=destination` | Warning: not yet supported | W060 |
| `spec.cloudquery.grpcPort` | Must be 1–65535 if provided | E062 |
| `spec.cloudquery.concurrency` | Must be > 0 if provided | E063 |
| `spec.runtime` | Required when `type=cloudquery` | E040 (existing) |
| `spec.runtime.image` | Required when `type=cloudquery` | E041 (existing) |
| `spec.outputs` | NOT required for `type=cloudquery` | — (skip E003) |
