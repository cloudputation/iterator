#!/bin/bash
# Read the CHANGELOG.md file and extract the content for the most recent version.
# It include only the last changes for the current release.
CHANGELOG_CONTENT=$(awk '/^## / {n++} n == 2 {exit} {if(n > 0) print}' CHANGELOG.md)
echo "${CHANGELOG_CONTENT}"
