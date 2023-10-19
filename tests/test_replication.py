from http_utility import append_message, get_messages, block_replication


def test_basic_replication(primary_url, secondary1_url, secondary2_url) -> None:
    """simple test to ensure that whole system works as expected from 1 iteration"""
    # GIVEN
    messages = ["msg 1", "msg 2", "msg 3", "msg 4", "msg 5"]

    # WHEN
    for message in messages:
        is_added = append_message(primary_url, message, 3)
        assert is_added, "Failed to append message: " + message

    messages_primary = get_messages(primary_url)
    messages_secondary1 = get_messages(secondary1_url)
    messages_secondary2 = get_messages(secondary2_url)

    # THEN
    assert messages == messages_primary, "Incorrect messages in PRIMARY storage"
    assert messages == messages_secondary1, "Incorrect messages in SECONDARY_1 storage"
    assert messages == messages_secondary2, "Incorrect messages in SECONDARY_2 storage"


def test_inconsistency_and_eventual_consistency_during_replication_with_w_2(primary_url, secondary1_url,
                                                                            secondary2_url) -> None:
    """
    Emulate replicas inconsistency (and eventual consistency) with the master
    by introducing the artificial delay on the ONE secondary node.
    In this case, the master and secondary should temporarily return different messages lists.

    After delay on the secondary node, messages lists must be the same on all nodes
    """
    # GIVEN
    messages = ["msg 1", "msg 2", "msg 3", "msg 4", "msg 5"]

    # WHEN
    is_blocked = block_replication(secondary2_url, True)
    assert is_blocked

    for message in messages:
        is_added = append_message(primary_url, message, 2)
        assert is_added, "Failed to append message: " + message

    messages_primary = get_messages(primary_url)
    messages_secondary1 = get_messages(secondary1_url)
    messages_secondary2 = get_messages(secondary2_url)

    # THEN
    assert messages == messages_primary, "Incorrect messages in PRIMARY storage"
    assert messages == messages_secondary1, "Incorrect messages in SECONDARY_1 storage"

    assert [] == messages_secondary2, "Incorrect messages in SECONDARY_2 storage"

    # WHEN
    is_unblocked = block_replication(secondary2_url, False)
    assert is_unblocked

    messages_secondary2 = get_messages(secondary2_url)
    # THEN
    assert messages == messages_secondary2, "Incorrect messages in SECONDARY_2 storage"


def test_inconsistency_and_eventual_consistency_during_replication_with_w_1(primary_url, secondary1_url,
                                                                            secondary2_url) -> None:
    """
    Emulate replicas inconsistency (and eventual consistency) with the master
    by introducing the artificial delay on BOTH secondaries.
    In this case, the master and secondaries should temporarily return different messages lists.

    After delay on the secondary nodes, messages lists must be the same on all nodes
    """
    # GIVEN
    messages = ["msg 1", "msg 2", "msg 3", "msg 4", "msg 5"]

    # WHEN
    is_blocked = block_replication(secondary1_url, True)
    assert is_blocked
    is_blocked = block_replication(secondary2_url, True)
    assert is_blocked

    for message in messages:
        is_added = append_message(primary_url, message, 1)
        assert is_added, "Failed to append message: " + message

    messages_primary = get_messages(primary_url)
    messages_secondary1 = get_messages(secondary1_url)
    messages_secondary2 = get_messages(secondary2_url)

    # THEN
    assert messages == messages_primary, "Incorrect messages in PRIMARY storage"

    assert [] == messages_secondary1, "Incorrect messages in SECONDARY_1 storage"
    assert [] == messages_secondary2, "Incorrect messages in SECONDARY_2 storage"

    # WHEN
    is_unblocked = block_replication(secondary1_url, False)
    assert is_unblocked
    is_unblocked = block_replication(secondary2_url, False)
    assert is_unblocked

    messages_secondary1 = get_messages(secondary1_url)
    messages_secondary2 = get_messages(secondary2_url)
    # THEN
    assert messages == messages_secondary1, "Incorrect messages in SECONDARY_1 storage"
    assert messages == messages_secondary2, "Incorrect messages in SECONDARY_2 storage"
