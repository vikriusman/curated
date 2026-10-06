# syntax=docker/dockerfile:1.7
FROM webdevops/php-nginx:8.2-alpine

ARG CI3_VERSION=3.1.13
ARG CI3_SHA256=8cb7bea9db3aa6aaa028ec9793e65b21540ec21890505d8ef03addcba46f9d23

ENV WEB_DOCUMENT_ROOT=/app/public \
    PHP_DISPLAY_ERRORS=0 \
    PHP_UPLOAD_MAX_FILESIZE=5M \
    PHP_POST_MAX_SIZE=6M \
    UPLOAD_PATH=/data/uploads

# CI3 is fetched from the official release and checksum-verified.
# This repo only contains application code (application/), not a framework copy.
ADD --checksum=sha256:${CI3_SHA256} \
    https://github.com/bcit-ci/CodeIgniter/archive/refs/tags/${CI3_VERSION}.tar.gz /tmp/ci3.tar.gz

RUN set -eux; \
    mkdir -p /app/public; \
    tar -xzf /tmp/ci3.tar.gz -C /tmp; \
    src=/tmp/CodeIgniter-${CI3_VERSION}; \
    mv $src/system $src/application /app/; \
    mv $src/index.php /app/public/index.php; \
    rm -rf /tmp/ci3.tar.gz $src /app/application/controllers/Welcome.php /app/application/views/welcome_message.php; \
    # Front controller in public/, framework and app outside the document root.
    sed -i \
      -e "s#^\(\s*\)\$system_path = 'system';#\1\$system_path = '../system';#" \
      -e "s#^\(\s*\)\$application_folder = 'application';#\1\$application_folder = '../application';#" \
      -e "s#: 'development');#: 'production');#" \
      /app/public/index.php; \
    grep -q "system_path = '../system'" /app/public/index.php; \
    grep -q "application_folder = '../application'" /app/public/index.php; \
    grep -q ": 'production');" /app/public/index.php

COPY app/application/ /app/application/
COPY docker/nginx/20-uploads.conf /opt/docker/etc/nginx/vhost.common.d/20-uploads.conf

# S3 mount point. In a container without the mount (plain docker run) this is local disk.
RUN mkdir -p /data/uploads \
 && chown -R application:application /data/uploads /app/application/logs /app/application/cache
