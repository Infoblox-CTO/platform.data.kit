# Deployment Configuration Apps (dc_apps) Core Data Set

App inventory from the deployment-configurations repository, providing namespace and ownership metadata keyed by app ID.

## Source

- **Ref:** `https://github.com/Infoblox-CTO/deployment-configurations/blob/master/apps.yaml`
- **Structure:** Top-level key `apps:`, where each child key is an app ID (e.g. `my-service`). Each entry is an object with fields including `namespace`, `owner`, and `productOwner`.

### Retrieval Process

1. **Parse**: Load the YAML file and descend into the top-level `apps` key.
2. **Enumerate entries**: Each child key of `apps` is the app `id`. Iterate over all child keys.
3. **Extract fields**: For each entry, read `namespace`, `owner`, and `productOwner`. The `id` is the map key itself, not a field within the object.
4. **Normalize optional fields**: `owner` and `productOwner` may be absent on some entries. Treat absent fields as `null` / empty string as appropriate for your use case.

## Dataset Fields

| Field | Source path | Description |
|-------|-------------|-------------|
| `id` | key under `apps` | App identifier |
| `namespace` | `apps.<id>.namespace` | Kubernetes namespace the app is deployed into |
| `owner` | `apps.<id>.owner` | Engineering owner (team or individual) |
| `product_owner` | `apps.<id>.productOwner` | Product owner |

### Example File

```yaml
apps:
  my-service:
    namespace: my-namespace
    owner: some-team
    productOwner: some-person
```

