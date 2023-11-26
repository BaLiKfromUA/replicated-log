import threading
import time

from http_utility import get_messages, block_replication, send_messages, HEALTHCHECK_PERIOD, append_message


def test_inconsistency_and_eventual_consistency_during_replication_with_w_2(primary_url, secondary1_url,
                                                                            secondary2_url) -> None:
    """
    Emulate replicas inconsistency (and eventual consistency) with the master
    by introducing the artificial delay on the ONE secondary node.
    In this case, the master and secondary should temporarily return different messages lists.

    After delay on the secondary node, messages lists must be the same on all nodes
    """
    # GIVEN
    w = 2
    messages = ["msg 1", "msg 2", "msg 3", "msg 4", "msg 5"]

    # WHEN
    is_blocked = block_replication(secondary2_url, True)
    assert is_blocked

    send_messages(primary_url, messages, w)

    messages_primary = get_messages(primary_url)
    messages_secondary1 = get_messages(secondary1_url)
    messages_secondary2 = get_messages(secondary2_url)

    # THEN
    assert messages == messages_primary, "Incorrect messages in PRIMARY storage"
    assert messages == messages_secondary1, "Incorrect messages in SECONDARY_1 storage"
    # the master and secondary should temporarily return different messages lists.
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
    w = 1
    messages = ["msg 1", "msg 2", "msg 3", "msg 4", "msg 5"]

    # WHEN
    is_blocked = block_replication(secondary1_url, True)
    assert is_blocked
    is_blocked = block_replication(secondary2_url, True)
    assert is_blocked

    send_messages(primary_url, messages, w)

    messages_primary = get_messages(primary_url)
    messages_secondary1 = get_messages(secondary1_url)
    messages_secondary2 = get_messages(secondary2_url)

    # THEN
    assert messages == messages_primary, "Incorrect messages in PRIMARY storage"
    # the master and secondaries should temporarily return different messages lists.
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


def test_if_primary_is_blocked_in_case_of_w_less_then_number_of_available_nodes(primary_url, secondary2_url) -> None:
    # GIVEN
    w = 3
    messages = ["test message"]

    # WHEN
    is_blocked = block_replication(secondary2_url, True)
    assert is_blocked

    adding_thread = threading.Thread(target=send_messages, args=(primary_url, messages, w))

    adding_thread.start()
    # hack to trigger current_thread.yield()
    # taken from https://stackoverflow.com/a/787810
    time.sleep(0.2)

    messages_secondary2 = get_messages(secondary2_url)

    # THEN
    assert [] == messages_secondary2, "Incorrect messages in SECONDARY_2 storage"
    assert adding_thread.is_alive()  # request is still blocked

    # WHEN
    is_unblocked = block_replication(secondary2_url, False)
    assert is_unblocked

    adding_thread.join()  # wait for primary unblocking
    messages_secondary2 = get_messages(secondary2_url)

    # THEN
    assert messages == messages_secondary2, "Incorrect messages in SECONDARY_2 storage"
    assert not adding_thread.is_alive()


def test_if_primary_rejects_append_request_if_there_is_no_quorum(primary_url, secondary1_url, secondary2_url) -> None:
    """
    If there is no quorum the master should be switched into read-only mode
    and shouldn’t accept messages append requests and should return the appropriate message
    """
    # GIVEN
    w = 2
    msg = "No Quorum"

    # WHEN
    is_blocked = block_replication(secondary1_url, True)
    assert is_blocked

    is_blocked = block_replication(secondary2_url, True)
    assert is_blocked

    time.sleep(int(HEALTHCHECK_PERIOD) / 1000 * 1.5)  # wait till health will be updated

    # THEN
    is_appended = append_message(primary_url, msg, w)
    assert not is_appended  # rejected

    primary_msg = get_messages(primary_url)
    assert [] == primary_msg
