"""Generate dbt profiles.yml from DK Store environment variables.

This module is used as a Docker entrypoint for dbt transforms:

    dk-profiles generate          # writes profiles.yml to current dir
    dk-profiles generate -o /app  # writes to /app/profiles.yml

Or programmatically:

    from datakit.profiles import generate_profiles
    yaml_str = generate_profiles("my_project")
"""

from __future__ import annotations

import argparse
import sys
from pathlib import Path
from urllib.parse import urlparse

import yaml

from datakit import stores


def generate_profiles(project_name: str, target: str = "dk") -> str:
    """Generate a dbt profiles.yml string from DK Store env vars.

    Finds all stores of type 'postgres' or 'postgresql' and uses the first one
    (dbt connects to a single database per run).

    Returns the YAML string for profiles.yml.
    """
    all_stores = stores.all()

    # Find the postgres store.
    pg_store = None
    for store in all_stores.values():
        if store.type in ("postgres", "postgresql"):
            pg_store = store
            break

    if pg_store is None:
        raise RuntimeError(
            "No postgres store found in environment. "
            "Ensure DK_STORE_DSN_* and DK_STORE_TYPE_* are set."
        )

    output = _dsn_to_dbt_output(pg_store.dsn)

    profiles = {
        project_name: {
            "target": target,
            "outputs": {
                target: output,
            },
        },
    }

    return yaml.dump(profiles, default_flow_style=False, sort_keys=False)


def _dsn_to_dbt_output(dsn: str) -> dict:
    """Parse a postgresql:// DSN into a dbt output config dict."""
    u = urlparse(dsn)

    host = u.hostname or "localhost"
    port = u.port or 5432
    dbname = u.path.lstrip("/") if u.path else ""
    user = u.username or ""
    password = u.password or ""

    # Check for schema in query params.
    from urllib.parse import parse_qs

    params = parse_qs(u.query)
    schema = params.get("schema", ["public"])[0]

    return {
        "type": "postgres",
        "host": host,
        "port": port,
        "user": user,
        "pass": password,
        "dbname": dbname,
        "schema": schema,
        "threads": 4,
    }


def main():
    """CLI entrypoint: dk-profiles generate."""
    parser = argparse.ArgumentParser(
        prog="dk-profiles",
        description="Generate dbt profiles.yml from DK Store environment variables",
    )
    sub = parser.add_subparsers(dest="command")

    gen = sub.add_parser("generate", help="Generate profiles.yml")
    gen.add_argument(
        "-o",
        "--output-dir",
        default=".",
        help="Directory to write profiles.yml (default: current dir)",
    )
    gen.add_argument(
        "--project",
        default=None,
        help="dbt project name (default: read from dbt_project.yml)",
    )
    gen.add_argument(
        "--target",
        default="dk",
        help="dbt target name (default: dk)",
    )

    args = parser.parse_args()

    if args.command != "generate":
        parser.print_help()
        sys.exit(1)

    # Determine project name.
    project_name = args.project
    if project_name is None:
        project_name = _read_project_name(args.output_dir)

    content = generate_profiles(project_name, args.target)

    out_path = Path(args.output_dir) / "profiles.yml"
    out_path.write_text(content)
    print(f"Generated {out_path}")


def _read_project_name(directory: str) -> str:
    """Read the project name from dbt_project.yml if it exists."""
    project_file = Path(directory) / "dbt_project.yml"
    if project_file.exists():
        with open(project_file) as f:
            project = yaml.safe_load(f)
            if isinstance(project, dict) and "name" in project:
                return project["name"]
    return "dk_project"


if __name__ == "__main__":
    main()
