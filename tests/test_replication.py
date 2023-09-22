import os

from http_utility import append_message, get_messages

PRIMARY_URL = os.getenv("PRIMARY_URL")
SECONDARY1_URL = os.getenv("SECONDARY1_URL")
SECONDARY2_URL = os.getenv("SECONDARY2_URL")


def test_basic_replication() -> None:
    # GIVEN
    message = "test"
    # WHEN
    is_added = append_message(PRIMARY_URL, message, 3)  # require ACK from master and two secondaries

    messages_primary = get_messages(PRIMARY_URL)
    messages_secondary1 = get_messages(SECONDARY1_URL)
    messages_secondary2 = get_messages(SECONDARY2_URL)
    # THEN
    assert is_added, "Failed to append a message"

    assert message in messages_primary, "No message in PRIMARY storage"
    assert message in messages_secondary1, "No message in SECONDARY_1 storage"
    assert message in messages_secondary2, "No message in SECONDARY_2 storage"
