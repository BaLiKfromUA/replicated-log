import requests


def get_messages(url: str) -> list[str]:
    resp = requests.get(url=url + "/api/v1/messages")
    assert resp.status_code == 200
    data = resp.json()
    print(data)
    assert data["messages"] is not None
    return data["messages"]


def append_message(url: str, message: str, w: int) -> bool:
    data = {
        "message": message,
        "w": w
    }
    headers = {'Content-type': 'application/json', 'Accept': 'text/plain'}
    resp = requests.post(url=url + "/api/v1/append", json=data, headers=headers)
    print(resp)
    return resp.status_code == 200


def clean_storage(url: str) -> bool:
    resp = requests.post(url=url + "/api/test/clean")
    print(resp)
    return resp.status_code == 200


def block_replication(url: str, block: bool) -> bool:
    data = {
        "enable": block
    }
    headers = {'Content-type': 'application/json', 'Accept': 'text/plain'}
    resp = requests.post(url=url + "/api/test/replication_block", json=data, headers=headers)
    print(resp)
    return resp.status_code == 200
