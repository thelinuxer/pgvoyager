#!/bin/bash
# Cleanup all test connections

curl -s http://localhost:5138/api/connections | jq -r '.[].id' | while read id; do
  echo "Deleting $id"
  curl -s -X DELETE "http://localhost:5138/api/connections/$id"
done

echo "Cleanup complete"
