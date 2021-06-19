#!/usr/bin/env bash

set -euf -o pipefail

go get -u github.com/tsenart/vegeta

# watch latencies distribution
echo "GET https://localhost.localdomain:1443/" |                          \
  vegeta attack -root-certs ../server/server.crt -duration=5s -rate 100 | \
  tee results.bin |                                                       \
  vegeta report

