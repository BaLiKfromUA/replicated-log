import threading
import time

from http_utility import send_messages, get_messages, block_replication, append_message


def test_basic_replication_example(primary_url, secondary1_url, secondary2_url) -> None:
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


def test_iteration3_example(primary_url, secondary2_url) -> None:
    """
    Self-check acceptance test:
    1) Start M + S1
    2) send (Msg1, W=1) - Ok
    3) send (Msg2, W=2) - Ok
    4) send (Msg3, W=3) - Wait
    5) send (Msg4, W=1) - Ok
    6) Start S2
    7) Check messages on S2 - [Msg1, Msg2, Msg3, Msg4]
    """

    # 1) stop S2
    is_blocked = block_replication(secondary2_url, True)
    assert is_blocked

    # 2) send (Msg1, W=1) - Ok
    is_added = append_message(primary_url, "Msg1", 1)
    assert is_added

    # 3) send (Msg2, W=2) - Ok
    is_added = append_message(primary_url, "Msg2", 1)
    assert is_added

    # 4) send (Msg3, W=3) - Wait
    adding_thread = threading.Thread(target=append_message, args=(primary_url, "Msg3", 3))

    adding_thread.start()
    # hack to trigger current_thread.yield()
    # taken from https://stackoverflow.com/a/787810
    time.sleep(0.2)

    # 5) send (Msg4, W=1) - Ok
    is_added = append_message(primary_url, "Msg4", 1)
    assert is_added

    # 6) Start S2
    is_unblocked = block_replication(secondary2_url, False)
    assert is_unblocked

    # 6.1) Wait that adding of Msg3 is finished
    adding_thread.join()
    assert not adding_thread.is_alive()

    # 7) Check messages on S2 - [Msg1, Msg2, Msg3, Msg4]
    messages_secondary2 = get_messages(secondary2_url)
    assert ["Msg1", "Msg2", "Msg3", "Msg4"] == messages_secondary2, "Incorrect messages in SECONDARY_2 storage"
