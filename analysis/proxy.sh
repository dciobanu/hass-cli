#!/bin/sh

# Check if homebrew is installed
if ! command -v brew &> /dev/null
then
    echo "Homebrew could not be found, please install it first."
    exit 1
fi

# Check if mitmproxy is installed
if ! command -v mitmproxy &> /dev/null
then
    echo "mitmproxy could not be found, installing it now..."
    brew install mitmproxy
else
    echo "mitmproxy is already installed."
fi

# Find the root of the repo
REPO_ROOT=$(git rev-parse --show-toplevel)

# Create the tmp folder if it doesn't exist
mkdir -p "$REPO_ROOT/tmp"

pushd "$REPO_ROOT/tmp" || exit 2

# Start mitmproxy with a specified port
mitmdump --mode reverse:http://192.168.5.23:8123 -p 8123 -s ../analysis/json_logger.py

popd || exit 2
