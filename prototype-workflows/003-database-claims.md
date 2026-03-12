# 003 — Database Claims

Inventory all database claims across clusters, enriched with ownership and database metadata. Supports filtering by database type, version, shape, app, team, VP, and cluster.

## Sources

### `k8s_database_claims`
- **Kind:** Source
- **Description:** Custom resource instances of `databaseclaims.persistance.atlas.infoblox.com` from Kubernetes. Captures declared database configuration per workload.
- **Connector:** Kubernetes API (CRD)
- **CRD:** `databaseclaims.persistance.atlas.infoblox.com`
- **Key fields:** `metadata.name`, `metadata.namespace`, `spec.type`, `spec.dbVersion`, `spec.shape`, `spec.appId`, `status.activeDB`, `status.error`

### `managed_teams`
- **Kind:** Source (Core Data Set)
- **Description:** Managed teams and group hierarchy from the bloxportal-catalog-metadata repository. Provides team membership and org structure.
- **Core Data Set:** [002-managed-teams](core-data-sets/002-managed-teams.md)

### `engineering_members`
- **Kind:** Source (Core Data Set)
- **Description:** Engineering organization members including name, email, team, and reporting chain (up to VP level).
- **Core Data Set:** [003-engineering-members](core-data-sets/003-engineering-members.md)

### `dc_apps`
- **Kind:** Source (Core Data Set)
- **Description:** App inventory from deployment-configurations, mapping app IDs to namespaces and owners. The `spec.appId` field on each database claim joins directly to `dc_apps.id`.
- **Core Data Set:** [001-dc-apps](core-data-sets/001-dc-apps.md)

## Transforms

### `database_claim_inventory`
- **Inputs:** `k8s_database_claims`
- **Description:** Normalize raw CRD data into a flat inventory of database claims with key database metadata fields extracted. `db_hostname` is sourced from `status.activeDB.connectionInfo.hostName`; `db_database` from `status.activeDB.connectionInfo.databaseName`.
- **Output fields:** `cluster`, `namespace`, `claim_name`, `app_id`, `db_type`, `db_version`, `shape`, `db_hostname`, `db_database`, `error`

### `claim_ownership`
- **Inputs:** `database_claim_inventory`, `managed_teams`, `engineering_members`, `dc_apps`
- **Description:** Resolve ownership of each database claim by joining `app_id` to `dc_apps` (for namespace and direct owner), then cross-referencing `managed_teams` for team membership and `engineering_members` for VP-level org hierarchy.
- **Output fields:** `cluster`, `namespace`, `claim_name`, `app_id`, `app_name`, `component_name`, `team`, `owner_ref`, `owner_email`, `vp_name`, `vp_email`, `db_type`, `db_version`, `shape`, `db_hostname`, `db_database`, `error`

### `database_claims_filtered`
- **Inputs:** `claim_ownership`
- **Description:** Parameterized filter transform for scoping the database claims report.
- **Parameters:**
  - `cluster` (optional) — filter to a specific cluster or list of clusters
  - `db_type` (optional) — filter by database type (e.g. `postgres`, `aurora-postgresql`)
  - `db_version` (optional) — filter by database version
  - `shape` (optional) — filter by instance shape/size
  - `team` (optional) — filter to a specific team
  - `app_name` (optional) — filter to a specific app
  - `vp_name` (optional) — filter to claims under a specific VP
- **Output fields:** same as `claim_ownership`

## Destinations

### `database_claims_report`
- **Kind:** Destination
- **Description:** Full inventory of database claims with ownership and database metadata, suitable for dashboarding, capacity planning, or CSV export.
- **Fields:** `cluster`, `namespace`, `claim_name`, `app_id`, `app_name`, `team`, `owner_email`, `vp_name`, `vp_email`, `db_type`, `db_version`, `shape`, `db_hostname`, `db_database`, `error`
