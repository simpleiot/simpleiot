#!/bin/bash

count=0

while true; do
  echo "running test ..."
  go clean -testcache
  if ! go test ./client/; then
    echo "test failed at count: $count"
    break
  fi
  count=$((count + 1))
  echo "count: $count"
done
