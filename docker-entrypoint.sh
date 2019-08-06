#!/bin/sh

# su-exec to requested user, if service cannot run exec will fail.
# Credits to MinIO project.
docker_switch_user() {
    if [ ! -z "${STANDARDFILE_USERNAME}" ] && [ ! -z "${STANDARDFILE_GROUPNAME}" ]; then
        addgroup -S "$STANDARDFILE_GROUPNAME" >/dev/null 2>&1 && \
            adduser -S -G "$STANDARDFILE_GROUPNAME" "$STANDARDFILE_USERNAME" >/dev/null 2>&1

        chown -R "${STANDARDFILE_USERNAME}:${STANDARDFILE_GROUPNAME}" $DATABASE_PATH

        exec su-exec "${STANDARDFILE_USERNAME}:${STANDARDFILE_GROUPNAME}" "$@"
    else
        # fallback
        exec "$@"
    fi
}

docker_switch_user "$@"