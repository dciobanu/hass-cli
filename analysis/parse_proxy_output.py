#!/usr/bin/env python

# This script analyses the output from a proxy server and extracts relevant information.
# It is designed to work with mitmproxy output files.
# It extracts unique URLs accessed and the corresponding HTTP methods used,
# request and response headers (masking the Authorization header), and bodies.
# It's also finding unique websocket messages sent to the server and responses received.
# Matching requests and responses based on flow IDs (.id field in the messages)

import json
import sys
from collections import defaultdict
from pathlib import Path

INPUT_FILE = "../tmp/ha_traffic.jsonl"
OUTPUT_FILE = "../tmp/parsed_proxy_output.txt"


def mask_sensitive_headers(headers: dict) -> dict:
    """Mask sensitive header values like Authorization."""
    masked = {}
    sensitive_keys = {"authorization", "cookie", "set-cookie", "x-api-key"}
    for key, value in headers.items():
        if key.lower() in sensitive_keys:
            masked[key] = "***MASKED***"
        else:
            masked[key] = value
    return masked


def mask_sensitive_body(body: str) -> str:
    """Mask sensitive values in request/response bodies."""
    if not body:
        return body

    import re

    # Mask tokens that look like JWTs (header.payload.signature)
    body = re.sub(
        r'eyJ[A-Za-z0-9_-]*\.eyJ[A-Za-z0-9_-]*\.[A-Za-z0-9_-]*',
        '***JWT_TOKEN***',
        body
    )

    # Mask refresh tokens (long hex strings, typically 64+ chars)
    body = re.sub(
        r'refresh_token["\s:=]+([a-f0-9]{64,})',
        r'refresh_token": "***MASKED_REFRESH_TOKEN***',
        body,
        flags=re.IGNORECASE
    )

    # Mask access tokens in JSON
    body = re.sub(
        r'"access_token"\s*:\s*"[^"]*"',
        '"access_token": "***MASKED***"',
        body
    )

    return body


def parse_ws_content(content: str) -> dict | None:
    """Parse WebSocket message content as JSON."""
    if not content or content == "<binary>":
        return None
    try:
        return json.loads(content)
    except json.JSONDecodeError:
        return None


def format_json(obj, indent=2, max_length=500, mask_secrets=False) -> str:
    """Format JSON object, truncating if too long."""
    if obj is None:
        return "null"
    formatted = json.dumps(obj, indent=indent, ensure_ascii=False)
    if mask_secrets:
        formatted = mask_sensitive_body(formatted)
    if len(formatted) > max_length:
        return formatted[:max_length] + "\n... (truncated)"
    return formatted


def main():
    input_path = Path(__file__).parent / INPUT_FILE
    output_path = Path(__file__).parent / OUTPUT_FILE

    if not input_path.exists():
        print(f"Error: Input file not found: {input_path}")
        sys.exit(1)

    # Data structures for collecting information
    http_endpoints = defaultdict(lambda: {
        "methods": set(),
        "requests": [],
        "responses": []
    })

    ws_message_types = defaultdict(lambda: {
        "samples": [],
        "count": 0
    })

    ws_requests = {}  # id -> request message
    ws_responses = {}  # id -> response message
    ws_matched_pairs = []  # (request, response) pairs

    # Parse the JSONL file
    with open(input_path, "r") as f:
        for line_num, line in enumerate(f, 1):
            line = line.strip()
            if not line:
                continue

            try:
                record = json.loads(line)
            except json.JSONDecodeError as e:
                print(f"Warning: Failed to parse line {line_num}: {e}")
                continue

            if record.get("type") == "websocket":
                # WebSocket message
                content = parse_ws_content(record.get("content"))
                direction = record.get("direction")

                if content and isinstance(content, dict):
                    msg_type = content.get("type", "unknown")
                    msg_id = content.get("id")

                    # Track message types
                    ws_message_types[msg_type]["count"] += 1
                    if len(ws_message_types[msg_type]["samples"]) < 3:
                        ws_message_types[msg_type]["samples"].append(content)

                    # Track request/response pairs by ID
                    if msg_id is not None:
                        if direction == "client":
                            ws_requests[msg_id] = content
                        else:
                            ws_responses[msg_id] = content
                            # Try to match with request
                            if msg_id in ws_requests:
                                ws_matched_pairs.append((ws_requests[msg_id], content))
            else:
                # HTTP request/response
                url = record.get("url", "")
                method = record.get("method", "")

                # Extract path from URL (remove host)
                if "://" in url:
                    path = "/" + url.split("://", 1)[1].split("/", 1)[-1]
                else:
                    path = url

                http_endpoints[path]["methods"].add(method)

                # Store request details
                request_info = {
                    "method": method,
                    "headers": mask_sensitive_headers(record.get("request_headers", {})),
                    "body": record.get("request_body")
                }

                response_info = {
                    "status_code": record.get("status_code"),
                    "headers": mask_sensitive_headers(record.get("response_headers", {})),
                    "body": record.get("response_body")
                }

                # Keep only unique request/response combinations (first occurrence)
                req_sig = f"{method}:{json.dumps(request_info['body'], sort_keys=True) if request_info['body'] else ''}"
                existing_sigs = [f"{r['method']}:{json.dumps(r['body'], sort_keys=True) if r['body'] else ''}"
                                for r in http_endpoints[path]["requests"]]

                if req_sig not in existing_sigs:
                    http_endpoints[path]["requests"].append(request_info)
                    http_endpoints[path]["responses"].append(response_info)

    # Generate output
    output_lines = []

    # Section 1: HTTP Endpoints
    output_lines.append("=" * 80)
    output_lines.append("HTTP ENDPOINTS")
    output_lines.append("=" * 80)
    output_lines.append("")

    for path in sorted(http_endpoints.keys()):
        info = http_endpoints[path]
        methods = ", ".join(sorted(info["methods"]))
        output_lines.append(f"### {methods} {path}")
        output_lines.append("")

        for i, (req, resp) in enumerate(zip(info["requests"], info["responses"])):
            if len(info["requests"]) > 1:
                output_lines.append(f"**Example {i + 1}:**")

            output_lines.append(f"**Request Headers:**")
            output_lines.append("```json")
            output_lines.append(format_json(req["headers"]))
            output_lines.append("```")

            if req["body"]:
                output_lines.append(f"**Request Body:**")
                output_lines.append("```json")
                masked_body = mask_sensitive_body(req["body"])
                try:
                    body = json.loads(masked_body)
                    output_lines.append(format_json(body, max_length=1000))
                except:
                    output_lines.append(masked_body[:1000])
                output_lines.append("```")

            output_lines.append(f"**Response Status:** {resp['status_code']}")

            if resp["body"]:
                output_lines.append(f"**Response Body:**")
                output_lines.append("```json")
                masked_body = mask_sensitive_body(resp["body"])
                try:
                    body = json.loads(masked_body)
                    output_lines.append(format_json(body, max_length=1000))
                except:
                    output_lines.append(masked_body[:1000])
                output_lines.append("```")

            output_lines.append("")

        output_lines.append("-" * 40)
        output_lines.append("")

    # Section 2: WebSocket Message Types
    output_lines.append("")
    output_lines.append("=" * 80)
    output_lines.append("WEBSOCKET MESSAGE TYPES")
    output_lines.append("=" * 80)
    output_lines.append("")

    for msg_type in sorted(ws_message_types.keys()):
        info = ws_message_types[msg_type]
        output_lines.append(f"### {msg_type} (count: {info['count']})")
        output_lines.append("")

        for i, sample in enumerate(info["samples"]):
            if len(info["samples"]) > 1:
                output_lines.append(f"**Sample {i + 1}:**")
            output_lines.append("```json")
            output_lines.append(format_json(sample, max_length=800, mask_secrets=True))
            output_lines.append("```")
            output_lines.append("")

        output_lines.append("-" * 40)
        output_lines.append("")

    # Section 3: WebSocket Request/Response Pairs
    output_lines.append("")
    output_lines.append("=" * 80)
    output_lines.append("WEBSOCKET REQUEST/RESPONSE PAIRS")
    output_lines.append("=" * 80)
    output_lines.append("")

    # Group pairs by request type
    pairs_by_type = defaultdict(list)
    for req, resp in ws_matched_pairs:
        req_type = req.get("type", "unknown")
        pairs_by_type[req_type].append((req, resp))

    for req_type in sorted(pairs_by_type.keys()):
        pairs = pairs_by_type[req_type]
        output_lines.append(f"### {req_type} ({len(pairs)} occurrences)")
        output_lines.append("")

        # Show up to 2 examples per type
        for i, (req, resp) in enumerate(pairs[:2]):
            if len(pairs) > 1:
                output_lines.append(f"**Example {i + 1}:**")

            output_lines.append("**Request:**")
            output_lines.append("```json")
            output_lines.append(format_json(req, max_length=600, mask_secrets=True))
            output_lines.append("```")

            output_lines.append("**Response:**")
            output_lines.append("```json")
            output_lines.append(format_json(resp, max_length=600, mask_secrets=True))
            output_lines.append("```")
            output_lines.append("")

        output_lines.append("-" * 40)
        output_lines.append("")

    # Section 4: Summary Statistics
    output_lines.append("")
    output_lines.append("=" * 80)
    output_lines.append("SUMMARY")
    output_lines.append("=" * 80)
    output_lines.append("")
    output_lines.append(f"Total unique HTTP endpoints: {len(http_endpoints)}")
    output_lines.append(f"Total unique WebSocket message types: {len(ws_message_types)}")
    output_lines.append(f"Total matched WebSocket request/response pairs: {len(ws_matched_pairs)}")
    output_lines.append("")

    output_lines.append("**HTTP Endpoints by Method:**")
    method_counts = defaultdict(int)
    for info in http_endpoints.values():
        for method in info["methods"]:
            method_counts[method] += 1
    for method, count in sorted(method_counts.items()):
        output_lines.append(f"  {method}: {count}")
    output_lines.append("")

    output_lines.append("**WebSocket Message Types (client -> server):**")
    client_types = [t for t, info in ws_message_types.items()
                    if any(s.get("id") for s in info["samples"])]
    for msg_type in sorted(client_types):
        output_lines.append(f"  - {msg_type}")
    output_lines.append("")

    output_lines.append("**WebSocket Message Types (server -> client):**")
    server_types = ["result", "event", "auth_required", "auth_ok", "pong"]
    for msg_type in sorted(ws_message_types.keys()):
        if msg_type in server_types:
            output_lines.append(f"  - {msg_type}")

    # Write output
    output_content = "\n".join(output_lines)
    output_path.parent.mkdir(parents=True, exist_ok=True)
    with open(output_path, "w") as f:
        f.write(output_content)

    print(f"Output written to: {output_path}")
    print(f"\nQuick Summary:")
    print(f"  HTTP endpoints: {len(http_endpoints)}")
    print(f"  WebSocket message types: {len(ws_message_types)}")
    print(f"  Matched WS pairs: {len(ws_matched_pairs)}")


if __name__ == "__main__":
    main()
