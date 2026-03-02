# K8s API Group: `datakit.infoblox.dev`

**Feature**: 016-rename-cli-dk | **Date**: 2026-03-01

## API Group

All DataKit CRDs use the API group `datakit.infoblox.dev`.

## API Version

`datakit.infoblox.dev/v1alpha1`

## CRDs

| CRD | Full Name |
|-----|-----------|
| PackageDeployment | `packagedeployments.datakit.infoblox.dev` |
| Cell | `cells.datakit.infoblox.dev` |
| Store | `stores.datakit.infoblox.dev` |

## RBAC

Controller RBAC rules reference the API group:

```yaml
- apiGroups: ["datakit.infoblox.dev"]
  resources: ["packagedeployments", "cells", "stores"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
```

## Leader Election

```yaml
leaderElectionID: "dk-controller.datakit.infoblox.dev"
```

## No Backward Compatibility

This replaces the previous `dp.io` API group. No migration path is provided. The project is pre-production — existing clusters with `dp.io` CRDs should delete old CRDs and re-apply. This is a clean break per the constitution's pre-production compatibility policy.
