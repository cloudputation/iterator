#!/bin/bash

# Read the CHANGELOG.md file and extract the content for the most recent version.
# It include only the last changes for the current release.

CHANGELOG_CONTENT=$(awk '/^## / {n++} n == 2 {exit} {if(n > 0) print}' CHANGELOG.md)
# echo "Including change logs for current release"
echo "${CHANGELOG_CONTENT}"
# echo
# echo
# echo
# echo
# echo
# echo
CHANGELOG_CONTENT="${CHANGELOG_CONTENT//'%'/'%25'}"
CHANGELOG_CONTENT="${CHANGELOG_CONTENT//$'\n'/'%0A'}"
CHANGELOG_CONTENT="${CHANGELOG_CONTENT//$'\r'/'%0D'}"
# echo "${CHANGELOG_CONTENT}"
