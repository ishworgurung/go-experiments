#!/usr/bin/env bash

set -euf -o pipefail

openssl req -new -subj "/C=AU/CN=localhost.localdomain"       \
  -addext "subjectAltName = DNS:localhost.localdomain"        \
  -addext "certificatePolicies = 1.2.3.4"                     \
  -newkey rsa:2048 -x509 -sha256 -days 365 -keyout server.key \
  -out server.crt -nodes
