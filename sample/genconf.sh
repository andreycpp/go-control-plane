#!/bin/bash
set -euo pipefail

# Helper script that generates config with given number of listeners.
#Can be used to compare load times via files vs xDS.

NUM_ENTRIES=${1:-10000}
PORT_START=10000
OUT_FILE="static.yaml"

cat << EOF > $OUT_FILE
static_resources:
  clusters:
  - name: some_service
    connect_timeout: 0.25s
    type: STATIC
    lb_policy: ROUND_ROBIN
    load_assignment:
      cluster_name: some_service
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: 127.0.0.1
                port_value: 1234
  listeners:
EOF

for i in `seq 1 $NUM_ENTRIES`; do
  cat << EOF >> $OUT_FILE
  - name: listener_${i}
    address:
      socket_address: { address: 127.0.0.1, port_value: $((PORT_START + i)) }
    filter_chains:
    - filters:
      - name: envoy.filters.network.http_connection_manager
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
          stat_prefix: ingress_http
          codec_type: AUTO
          route_config:
            name: local_route
            virtual_hosts:
            - name: local_service
              domains: ["*"]
              routes:
              - match: { prefix: "/" }
                route: { cluster: some_service }
          http_filters:
          - name: envoy.filters.http.router
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.filters.http.router.v3.Router
EOF
  done

