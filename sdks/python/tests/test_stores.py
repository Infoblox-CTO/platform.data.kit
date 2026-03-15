import os

import pytest

from datakit import stores


@pytest.fixture(autouse=True)
def clean_env(monkeypatch):
    """Remove any DK_STORE_* env vars before each test."""
    for key in list(os.environ):
        if key.startswith("DK_STORE_"):
            monkeypatch.delenv(key)


def test_get_store(monkeypatch):
    monkeypatch.setenv("DK_STORE_DSN_WAREHOUSE", "postgresql://user:pass@host:5432/db")
    monkeypatch.setenv("DK_STORE_TYPE_WAREHOUSE", "postgres")

    store = stores.get("warehouse")
    assert store.dsn == "postgresql://user:pass@host:5432/db"
    assert store.type == "postgres"
    assert store.name == "warehouse"


def test_get_store_kebab_name(monkeypatch):
    monkeypatch.setenv("DK_STORE_DSN_LAKE_RAW", "s3://my-bucket?region=us-east-1")
    monkeypatch.setenv("DK_STORE_TYPE_LAKE_RAW", "s3")

    store = stores.get("lake-raw")
    assert store.dsn == "s3://my-bucket?region=us-east-1"
    assert store.type == "s3"


def test_get_store_not_found():
    with pytest.raises(KeyError, match="Store 'missing' not found"):
        stores.get("missing")


def test_all_stores(monkeypatch):
    monkeypatch.setenv("DK_STORE_DSN_WAREHOUSE", "postgresql://host/db")
    monkeypatch.setenv("DK_STORE_TYPE_WAREHOUSE", "postgres")
    monkeypatch.setenv("DK_STORE_DSN_LAKE_RAW", "s3://bucket")
    monkeypatch.setenv("DK_STORE_TYPE_LAKE_RAW", "s3")

    result = stores.all()
    assert len(result) == 2
    assert "warehouse" in result
    assert "lake-raw" in result
    assert result["warehouse"].type == "postgres"
    assert result["lake-raw"].type == "s3"


def test_all_stores_empty():
    result = stores.all()
    assert result == {}
