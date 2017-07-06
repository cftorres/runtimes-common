#!/bin/bash
test=$(go run main.go iDiff gcr.io/google_containers/busybox:1.24 gcr.io/google_containers/busybox:latest dir)
actual=$(cat 'test_images/busybox_diff.json' | jq '.')

if [ "test" != "actual" ]; then
	echo "iDiff output is not as expected"
	exit 1
fi