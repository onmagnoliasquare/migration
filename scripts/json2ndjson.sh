#!/bin/bash

cat test.json | jq -c '.[]' > test.ndjson
