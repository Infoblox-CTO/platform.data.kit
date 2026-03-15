import os

import pytest
import yaml

from datakit.profiles import generate_profiles, _dsn_to_dbt_output


@pytest.fixture(autouse=True)
def clean_env(monkeypatch):
    for key in list(os.environ):
        if key.startswith("DK_STORE_"):
            monkeypatch.delenv(key)


def test_generate_profiles(monkeypatch):
    monkeypatch.setenv("DK_STORE_DSN_WAREHOUSE", "postgresql://app:secret@prod-db:5432/mydb")
    monkeypatch.setenv("DK_STORE_TYPE_WAREHOUSE", "postgres")

    result = generate_profiles("my_project")
    parsed = yaml.safe_load(result)

    assert "my_project" in parsed
    assert parsed["my_project"]["target"] == "dk"
    output = parsed["my_project"]["outputs"]["dk"]
    assert output["type"] == "postgres"
    assert output["host"] == "prod-db"
    assert output["port"] == 5432
    assert output["user"] == "app"
    assert output["pass"] == "secret"
    assert output["dbname"] == "mydb"
    assert output["threads"] == 4


def test_generate_profiles_no_postgres(monkeypatch):
    monkeypatch.setenv("DK_STORE_DSN_EVENTS", "kafka://broker:9092/topic")
    monkeypatch.setenv("DK_STORE_TYPE_EVENTS", "kafka")

    with pytest.raises(RuntimeError, match="No postgres store found"):
        generate_profiles("my_project")


def test_dsn_to_dbt_output():
    output = _dsn_to_dbt_output("postgresql://user:pass@host:5432/db?schema=analytics")
    assert output["type"] == "postgres"
    assert output["host"] == "host"
    assert output["port"] == 5432
    assert output["user"] == "user"
    assert output["pass"] == "pass"
    assert output["dbname"] == "db"
    assert output["schema"] == "analytics"


def test_dsn_to_dbt_output_defaults():
    output = _dsn_to_dbt_output("postgresql://host/db")
    assert output["host"] == "host"
    assert output["port"] == 5432
    assert output["schema"] == "public"
    assert output["user"] == ""
    assert output["pass"] == ""
