#!/bin/sh

set -u -e

CHANGELOG=CHANGELOG.md
FOUND=0

TAG=$1
TAG_BASE=$(echo "${TAG}" | cut -d- -f1)

stringContain() {
	case $1 in
	$2*)
		return 0
		;;
	*)
		return 1
		;;
	esac
}

while IFS= read -r LINE; do
	case "$FOUND" in
	0)
		if stringContain "${LINE}" "## [${TAG_BASE}"; then
			echo "${LINE}"
			FOUND=$((FOUND + 1))
		fi
		;;
	1)
		if stringContain "${LINE}" "## ["; then
			FOUND=$((FOUND + 1))
		else
			echo "${LINE}"
		fi
		;;
	2)
		break
		;;
	esac
done <$CHANGELOG

if [ $FOUND -gt 0 ]; then
	exit 0
else
	echo "Version '${TAG_BASE}' is not found at ${CHANGELOG}."
	exit 1
fi
