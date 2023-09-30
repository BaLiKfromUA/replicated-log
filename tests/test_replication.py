import os

from http_utility import append_message, get_messages

PRIMARY_URL = os.getenv("PRIMARY_URL")
SECONDARY1_URL = os.getenv("SECONDARY1_URL")
SECONDARY2_URL = os.getenv("SECONDARY2_URL")


def test_basic_replication() -> None:
    # GIVEN
    messages = ["msg 1", "msg 2", "msg 3", "msg 4", "msg 5"]

    # WHEN
    for message in messages:
        is_added = append_message(PRIMARY_URL, message)
        assert is_added, "Failed to append message: " + message

    messages_primary = get_messages(PRIMARY_URL)
    messages_secondary1 = get_messages(SECONDARY1_URL)
    messages_secondary2 = get_messages(SECONDARY2_URL)

    # THEN
    assert messages == messages_primary, "Incorrect messages in PRIMARY storage"
    assert messages == messages_secondary1, "Incorrect messages in SECONDARY_1 storage"
    assert messages == messages_secondary2, "Incorrect messages in SECONDARY_2 storage"
