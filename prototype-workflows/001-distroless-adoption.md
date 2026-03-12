# 001 — Distroless Adoption

Track which workloads are using distroless base images and which are not, mapped to software ownership. Supports filtering and aggregation by app, team, VP, and cluster.

## Sources

### `wiz_images`
- **Kind:** Source
- **Description:** Container image data from Wiz, including base image layers and image digests.
- **Connector:** Wiz API

### `k8s_pods`
- **Kind:** Source
- **Description:** Pod inventory from Kubernetes, including namespace, image references, workload metadata, and cluster name.
- **Connector:** Kubernetes API

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
- **Description:** App inventory from deployment-configurations, mapping app IDs to namespaces and owners. Used to correlate Kubernetes namespaces to app ownership.
- **Core Data Set:** [001-dc-apps](core-data-sets/001-dc-apps.md)

### `distroless_base_images`
- **Kind:** Source
- **Description:** Canonical list of approved distroless base images (e.g. `gcr.io/distroless/*`, `cgr.dev/chainguard/*`).
- **Connector:** Static file or curated registry list

## Transforms

### `image_base_classification`
- **Inputs:** `wiz_images`, `distroless_base_images`
- **Description:** Join image data against the distroless base image list to classify each image as distroless or non-distroless.
- **Output fields:** `image_digest`, `image_name`, `base_image`, `is_distroless`

### `pod_image_join`
- **Inputs:** `k8s_pods`, `image_base_classification`
- **Description:** Join pod inventory with classified image data to determine distroless status per workload. Includes cluster name from pod metadata.
- **Output fields:** `cluster`, `namespace`, `pod_name`, `workload`, `image_digest`, `is_distroless`

### `workload_ownership`
- **Inputs:** `pod_image_join`, `managed_teams`, `engineering_members`, `dc_apps`
- **Description:** Resolve workload ownership by matching Kubernetes namespaces to `dc_apps` entries (for app ID and direct owner), then cross-referencing `managed_teams` for team membership and `engineering_members` for VP-level org hierarchy.
- **Output fields:** `cluster`, `workload`, `namespace`, `app_name`, `component_name`, `team`, `owner_ref`, `owner_email`, `vp_email`, `vp_name`, `is_distroless`, `base_image`

### `distroless_adoption_filtered`
- **Inputs:** `workload_ownership`
- **Description:** Parameterized filter transform for scoping the adoption report to a subset of workloads.
- **Parameters:**
  - `cluster` (optional) — filter to a specific cluster or list of clusters
  - `team` (optional) — filter to a specific team
  - `app_name` (optional) — filter to a specific app
  - `vp_name` (optional) — filter to workloads under a specific VP
- **Output fields:** same as `workload_ownership`

## Destinations

### `distroless_adoption_report`
- **Kind:** Destination
- **Description:** Summary report of distroless adoption by workload, component, and owner — suitable for dashboarding or CSV export.
- **Fields:** `cluster`, `workload`, `namespace`, `app_name`, `team`, `owner_email`, `vp_name`, `vp_email`, `is_distroless`, `base_image`
