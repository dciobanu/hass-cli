#!/usr/bin/env python

# This script analyses the output from a proxy server and extracts relevant information.
# It is designed to work with mitmproxy output files.
# It extracts unique URLs accessed and the corresponding HTTP methods used,
# request and response headers (masking the Authorization header), and bodies.
# It's also finding unique websocket messages sent to the server and responses received.
# Matching requests and responses based on flow IDs (.id field in the messages)

INPUT_FILE = "../tmp/ha_traffic.jsonl"
OUTPUT_FILE = "../tmp/parsed_proxy_output.txt"

