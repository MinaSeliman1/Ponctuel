#!/bin/sh
set -eu

create_topic() {
    topic="$1"
    if ! rpk topic describe "$topic" --brokers redpanda:9092 >/dev/null 2>&1; then
        rpk topic create "$topic" --brokers redpanda:9092 --partitions 1 --replicas 1
    fi
}

create_topic trip-updates
create_topic vehicle-positions
