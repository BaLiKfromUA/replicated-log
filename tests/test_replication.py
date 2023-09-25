from http_utility import append_message, get_messages


def test_basic_replication(primary_url, secondary1_url, secondary2_url) -> None:
    """simple test to ensure that whole system works as expected from 1 iteration"""
    # GIVEN
    message = "test"
    # WHEN
    is_added = append_message(primary_url, message, 3)  # require ACK from master and two secondaries

    messages_primary = get_messages(primary_url)
    messages_secondary1 = get_messages(secondary1_url)
    messages_secondary2 = get_messages(secondary2_url)
    # THEN
    assert is_added, "Failed to append a message"

    assert message in messages_primary, "No message in PRIMARY storage"
    assert message in messages_secondary1, "No message in SECONDARY_1 storage"
    assert message in messages_secondary2, "No message in SECONDARY_2 storage"
