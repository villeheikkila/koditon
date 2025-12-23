FROM postgres:18

ENV PG_CRON_VERSION=v1.6.7

# Install build dependencies and PostGIS
RUN apt-get update && apt-get install -y \
    build-essential \
    git \
    postgresql-server-dev-18 \
    libcurl4-openssl-dev \
    postgresql-18-postgis-3 \
    && rm -rf /var/lib/apt/lists/*

# Build and install pg_cron
RUN cd /tmp \
    && git clone https://github.com/citusdata/pg_cron.git \
    && cd pg_cron \
    && make \
    && make install \
    && cd / \
    && rm -rf /tmp/pg_cron

# Build and install pgmq
RUN cd /tmp \
    && git clone https://github.com/pgmq/pgmq.git \
    && cd pgmq/pgmq-extension \
    && make \
    && make install \
    && cd / \
    && rm -rf /tmp/pgmq

# Create initialization script to enable extensions
RUN mkdir -p /docker-entrypoint-initdb.d
COPY <<EOF /docker-entrypoint-initdb.d/init-extensions.sql
CREATE EXTENSION IF NOT EXISTS pg_cron;
CREATE EXTENSION IF NOT EXISTS pgmq;
CREATE SCHEMA IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS postgis SCHEMA postgis;
ALTER DATABASE postgres SET search_path TO public, postgis, "\$user";
SET search_path TO public, postgis;
EOF

EXPOSE 5432
