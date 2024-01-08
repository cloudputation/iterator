#!/usr/bin/dumb-init /bin/sh
# Copyright (c) Cloudputation, Inc.

set -e

# If the user is trying to run iterator directly with some arguments,
# then pass them to iterator.
# On alpine /bin/sh is busybox which supports the bashism below.
if [ "${1:0:1}" = '-' ]; then
	set -- /bin/iterator "$@"
fi

# If user is trying to run iterator with no arguments (daemon-mode),
# docker will run '/bin/sh -c /bin/${NAME}'. Check for the full command since
# running 'bin/sh' is a common pattern
if [ "$*" = '/bin/sh -c /bin/${NAME}' ]; then
	set -- /bin/iterator
fi

SF_CONFIG_DIR=/iterator/config

# Set the configuration directory
if [ "$1" = '/bin/iterator' ]; then
	shift
	set -- /bin/iterator agent #-config-dir="$SF_CONFIG_DIR" "$@"
fi

exec "$@"
