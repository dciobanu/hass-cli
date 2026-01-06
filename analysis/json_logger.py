import json
from mitmproxy import ctx

def response(flow):
    record = {
        "timestamp": flow.request.timestamp_start,
        "method": flow.request.method,
        "url": flow.request.pretty_url,
        "request_headers": dict(flow.request.headers),
        "request_body": flow.request.text if flow.request.text else None,
        "status_code": flow.response.status_code,
        "response_headers": dict(flow.response.headers),
        "response_body": flow.response.text if flow.response.text else None,
    }
    with open("ha_traffic.jsonl", "a") as f:
        f.write(json.dumps(record) + "\n")

def websocket_message(flow):
    msg = flow.websocket.messages[-1]
    record = {
        "timestamp": msg.timestamp,
        "type": "websocket",
        "url": flow.request.pretty_url,
        "direction": "client" if msg.from_client else "server",
        "content": msg.text if msg.is_text else "<binary>",
    }
    with open("ha_traffic.jsonl", "a") as f:
        f.write(json.dumps(record) + "\n")