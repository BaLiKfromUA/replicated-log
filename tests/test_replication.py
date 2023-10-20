from http_utility import send_messages, get_messages


def test_basic_replication(primary_url, secondary1_url, secondary2_url) -> None:
    """simple test to ensure that whole system works as expected from 1 iteration"""
    # GIVEN
    w = 3
    messages = ["msg 1", "msg 2", "msg 3", "msg 4", "msg 5"]

    # WHEN
    send_messages(primary_url, messages, w)

    messages_primary = get_messages(primary_url)
    messages_secondary1 = get_messages(secondary1_url)
    messages_secondary2 = get_messages(secondary2_url)

    # THEN
    assert messages == messages_primary, "Incorrect messages in PRIMARY storage"
    assert messages == messages_secondary1, "Incorrect messages in SECONDARY_1 storage"
    assert messages == messages_secondary2, "Incorrect messages in SECONDARY_2 storage"
