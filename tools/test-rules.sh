#!/bin/bash

count=0

while true; do
	echo "running test ..."
	go clean -testcache
	if ! go test -p=1 -race ./client -v -run TestDisabled; then
		echo "test failed at count: $count"
		break
	fi

	if ! go test -p=1 -race ./client -v -run TestDisabledCondition; then
		echo "test failed at count: $count"
		break
	fi
	count=$((count + 1))
	echo "count: $count"
done
