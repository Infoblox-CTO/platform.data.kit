# Managed Teams (managed_teams) Core Data Set

Managed teams data from the catalog and building a group hierarchy.

## Source

- **Ref:** `https://github.com/Infoblox-CTO/bloxportal-catalog-metadata/blob/main/catalog/all-managed-teams.yaml`
- **Structure:** The root `all-managed-teams.yaml` file has a list of `spec.targets` values, which are paths to other files (resolved from `catalog/`).

## Retrieval Process

1. **Parse root**: Load the YAML file and read all the `spec.targets` values.
2. **Load each target file**: For each path in `spec.targets`, open the YAML file and extract the dataset fields described below.
3. **Build a node map**: Index every loaded group by `metadata.name`. This produces a flat dictionary keyed on group name that can be used to resolve relationships.
4. **Resolve the hierarchy**

`spec.parent` uses Backstage entity reference syntax: `group:default/<name>`. Strip the `group:default/` prefix to get the bare name, then look it up in the node map to link each group to its parent.

Groups with no `spec.parent` (e.g. domain-group nodes) are the roots of their respective subtrees. A group whose parent is not present in `all-managed-teams.yaml` (e.g. `devops-feature-owners`) is treated as an external root.

## Dataset Fields

| Field           | Source Path              | Notes                                     |
| --------------- | ------------------------ | ----------------------------------------- |
| `name`        | `metadata.name`        | Required                                  |
| `namespace`   | `metadata.namespace`   | Typically `default`                     |
| `title`       | `metadata.title`       | Human-readable label                      |
| `description` | `metadata.description` |                                           |
| `tags`        | `metadata.tags`        | Optional list of strings                  |
| `parent`      | `spec.parent`          | Optional; format `group:default/<name>` |
| `children`    | `spec.children`        | List of child group references            |
| `members`     | `spec.members`         | List; format `user:okta/<login>`        |

### Field Notes

- **`spec.members`**: Members are listed as `user:okta/<okta-login>`. Cross-reference with `engineering-members` data set to map Okta logins to user information.

### Example Root File

```yaml
spec:
  targets:
    - ./groups/managed-teams/<team-name>.yaml
    - ./product-domains/<domain>/<domain>-group.yaml
    - ...
```

### Example Target File

```yaml
apiVersion: backstage.io/v1alpha1
kind: Group
metadata:
  name: assets-core-team
  namespace: default
  ...
  title: Assets Core Team
  description: Owners group for the assets-core-team - managed-team.
  tags:
  - business-service-owners
  - business-feature-owners
spec:
  ...
  parent: group:default/assets-domain-group
  children: []
  members:
  - user:okta/guptas
  - user:okta/bmuthusamy
  - user:okta/rmprabhakaran
```
