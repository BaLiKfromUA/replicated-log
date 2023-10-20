import os

import pytest

from http_utility import clean_storage

PRIMARY_URL = os.getenv("PRIMARY_URL")
SECONDARY1_URL = os.getenv("SECONDARY1_URL")
SECONDARY2_URL = os.getenv("SECONDARY2_URL")


@pytest.fixture
def primary_url() -> str:
    return PRIMARY_URL


@pytest.fixture
def secondary1_url() -> str:
    return SECONDARY1_URL


@pytest.fixture
def secondary2_url() -> str:
    return SECONDARY2_URL


@pytest.fixture(autouse=True)
def clean_up():
    yield
    # cleaning storages
    result = clean_storage(SECONDARY1_URL)
    assert result
    result = clean_storage(SECONDARY2_URL)
    assert result
    result = clean_storage(PRIMARY_URL)
    assert result
