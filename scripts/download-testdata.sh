#!/usr/bin/env bash
# Download the top ~50 Grafana dashboards for integration testing.
# Dashboards are saved to testdata/dashboards/.
# Skip download if the directory already has files.
set -euo pipefail

DIR="$(cd "$(dirname "$0")/.." && pwd)/testdata/dashboards"
mkdir -p "$DIR"

existing=$(find "$DIR" -name '*.json' 2>/dev/null | wc -l | tr -d ' ')
if [ "$existing" -ge 40 ]; then
  echo "testdata/dashboards/ already has $existing dashboards — skipping download"
  exit 0
fi

echo "Downloading Grafana dashboards to $DIR ..."

# Format: grafana_id:revision:filename
DASHBOARDS="
1860:37:node-exporter-full
315:3:kubernetes-cluster
6417:1:kubernetes-node-exporter
7249:1:kubernetes-cluster-prometheus
9628:8:postgresql
14031:1:mysql-overview
7362:5:mysql-performance
12708:1:nginx-ingress
11835:1:redis
2583:2:mongodb
10991:1:rabbitmq
7589:5:kafka-exporter
893:5:docker-container
15762:1:coredns
3070:3:etcd
13105:1:nginx-ingress-controller-nextgen
10000:1:kubernetes-cluster-autoscaler
747:2:blackbox-exporter
8588:1:kubernetes-deployment
5984:1:alertmanager
3662:2:prometheus-stats
12006:1:kubernetes-apiserver
1471:1:kube-state-metrics
455:2:prometheus
11074:1:node-exporter-server
763:3:redis-dashboard
4279:4:jvm-micrometer
3590:1:grafana-internals
4271:4:jvm-jolokia
10124:1:traefik
12740:1:kubernetes-pvc
8685:1:jenkins-performance
4358:1:minio
14981:1:kubernetes-networking
11455:4:haproxy
6671:1:loki-dashboard
14055:1:cadvisor
1617:1:prometheus-node
5228:1:zookeeper
928:3:cassandra
2322:3:memcached
12135:1:envoy
10427:2:kube-scheduler
10557:1:kubernetes-calico
16888:3:cert-manager
15760:1:kubewarden
11159:1:ceph-cluster
"

ok=0
fail=0
for entry in $DASHBOARDS; do
  IFS=: read -r id rev name <<< "$entry"
  out="$DIR/${name}.json"
  [ -f "$out" ] && { ok=$((ok+1)); continue; }
  url="https://grafana.com/api/dashboards/${id}/revisions/${rev}/download"
  if curl -sfL "$url" -o "$out" 2>/dev/null && [ "$(wc -c < "$out")" -gt 100 ]; then
    ok=$((ok+1))
  else
    rm -f "$out"
    fail=$((fail+1))
    echo "  skip: $name (ID $id rev $rev)"
  fi
done

echo "Done: $ok downloaded, $fail unavailable"
