"""Read DK Store connection info from environment variables.

The DataKit controller (or `dk run`) injects two env vars per store:

    DK_STORE_DSN_{NAME}   — connection string (e.g., postgresql://user:pass@host/db)
    DK_STORE_TYPE_{NAME}  — connector type (e.g., postgres, s3, kafka)

where {NAME} is the store name uppercased with hyphens replaced by underscores.

Usage:

    from datakit import stores

    warehouse = stores.get("warehouse")
    print(warehouse.dsn)   # "postgresql://..."
    print(warehouse.type)  # "postgres"

    for name, store in stores.all().items():
        print(f"{name}: {store.type}")
"""

from __future__ import annotations

import os
import re
from dataclasses import dataclass


@dataclass(frozen=True)
class Store:
    """A resolved store with its DSN and type."""

    name: str
    dsn: str
    type: str


_PREFIX_DSN = "DK_STORE_DSN_"
_PREFIX_TYPE = "DK_STORE_TYPE_"


def _env_name(store_name: str) -> str:
    """Convert a store name to its env var suffix (uppercase, hyphens → underscores)."""
    return store_name.upper().replace("-", "_").replace(".", "_")


def get(name: str) -> Store:
    """Get a store by its logical name.

    Raises KeyError if the store is not found in the environment.
    """
    env_suffix = _env_name(name)
    dsn = os.environ.get(_PREFIX_DSN + env_suffix)
    if dsn is None:
        raise KeyError(
            f"Store {name!r} not found. "
            f"Expected env var {_PREFIX_DSN}{env_suffix} to be set. "
            f"Is this transform running via `dk run` or the DK controller?"
        )
    store_type = os.environ.get(_PREFIX_TYPE + env_suffix, "")
    return Store(name=name, dsn=dsn, type=store_type)


def all() -> dict[str, Store]:
    """Discover all stores from the environment.

    Returns a dict mapping logical store name → Store.
    """
    stores: dict[str, Store] = {}
    for key, dsn in os.environ.items():
        if not key.startswith(_PREFIX_DSN):
            continue
        suffix = key[len(_PREFIX_DSN) :]
        store_type = os.environ.get(_PREFIX_TYPE + suffix, "")
        # Convert ENV_NAME back to a lowercase kebab name for display.
        name = suffix.lower().replace("_", "-")
        stores[name] = Store(name=name, dsn=dsn, type=store_type)
    return stores
