#!/usr/bin/env bats

load _helpers

@test "syncCatalog/Deployment: disabled by default" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/sync-catalog-deployment.yaml  \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "syncCatalog/Deployment: enable with global.enabled false" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/sync-catalog-deployment.yaml  \
      --set 'global.enabled=false' \
      --set 'syncCatalog.enabled=true' \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "true" ]
}

@test "syncCatalog/Deployment: disable with syncCatalog.enabled" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/sync-catalog-deployment.yaml  \
      --set 'syncCatalog.enabled=false' \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "syncCatalog/Deployment: disable with global.enabled" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/sync-catalog-deployment.yaml  \
      --set 'global.enabled=false' \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

#--------------------------------------------------------------------
# image

@test "syncCatalog/Deployment: image defaults to global.imageK8S" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/sync-catalog-deployment.yaml  \
      --set 'global.imageK8S=bar' \
      --set 'syncCatalog.enabled=true' \
      . | tee /dev/stderr |
      yq -r '.spec.template.spec.containers[0].image' | tee /dev/stderr)
  [ "${actual}" = "bar" ]
}

@test "syncCatalog/Deployment: image can be overridden with server.image" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/sync-catalog-deployment.yaml  \
      --set 'global.imageK8S=foo' \
      --set 'syncCatalog.enabled=true' \
      --set 'syncCatalog.image=bar' \
      . | tee /dev/stderr |
      yq -r '.spec.template.spec.containers[0].image' | tee /dev/stderr)
  [ "${actual}" = "bar" ]
}

#--------------------------------------------------------------------
# default sync

@test "syncCatalog/Deployment: default sync is true by default" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/sync-catalog-deployment.yaml  \
      --set 'syncCatalog.enabled=true' \
      . | tee /dev/stderr |
      yq -r '.spec.template.spec.containers[0].command | any(contains("-k8s-default-sync=true"))' | tee /dev/stderr)
  [ "${actual}" = "true" ]
}

@test "syncCatalog/Deployment: default sync can be turned off" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/sync-catalog-deployment.yaml  \
      --set 'syncCatalog.enabled=true' \
      --set 'syncCatalog.default=false' \
      . | tee /dev/stderr |
      yq -r '.spec.template.spec.containers[0].command | any(contains("-k8s-default-sync=false"))' | tee /dev/stderr)
  [ "${actual}" = "true" ]
}

#--------------------------------------------------------------------
# toConsul and toK8S

@test "syncCatalog/Deployment: bidirectional by default" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/sync-catalog-deployment.yaml  \
      --set 'syncCatalog.enabled=true' \
      . | tee /dev/stderr |
      yq '.spec.template.spec.containers[0].command | any(contains("-to-consul"))' | tee /dev/stderr)
  [ "${actual}" = "false" ]

  local actual=$(helm template \
      -x templates/sync-catalog-deployment.yaml  \
      --set 'syncCatalog.enabled=true' \
      . | tee /dev/stderr |
      yq '.spec.template.spec.containers[0].command | any(contains("-to-k8s"))' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "syncCatalog/Deployment: to-k8s only" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/sync-catalog-deployment.yaml  \
      --set 'syncCatalog.enabled=true' \
      --set 'syncCatalog.toConsul=false' \
      . | tee /dev/stderr |
      yq '.spec.template.spec.containers[0].command | any(contains("-to-consul=false"))' | tee /dev/stderr)
  [ "${actual}" = "true" ]

  local actual=$(helm template \
      -x templates/sync-catalog-deployment.yaml  \
      --set 'syncCatalog.enabled=true' \
      --set 'syncCatalog.toConsul=false' \
      . | tee /dev/stderr |
      yq '.spec.template.spec.containers[0].command | any(contains("-to-k8s"))' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "syncCatalog/Deployment: to-consul only" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/sync-catalog-deployment.yaml  \
      --set 'syncCatalog.enabled=true' \
      --set 'syncCatalog.toK8S=false' \
      . | tee /dev/stderr |
      yq '.spec.template.spec.containers[0].command | any(contains("-to-k8s=false"))' | tee /dev/stderr)
  [ "${actual}" = "true" ]

  local actual=$(helm template \
      -x templates/sync-catalog-deployment.yaml  \
      --set 'syncCatalog.enabled=true' \
      --set 'syncCatalog.toK8S=false' \
      . | tee /dev/stderr |
      yq '.spec.template.spec.containers[0].command | any(contains("-to-consul"))' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

#--------------------------------------------------------------------
# k8sPrefix

@test "syncCatalog/Deployment: no k8sPrefix by default" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/sync-catalog-deployment.yaml  \
      --set 'syncCatalog.enabled=true' \
      . | tee /dev/stderr |
      yq '.spec.template.spec.containers[0].command | any(contains("-k8s-service-prefix"))' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "syncCatalog/Deployment: can specify k8sPrefix" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/sync-catalog-deployment.yaml  \
      --set 'syncCatalog.enabled=true' \
      --set 'syncCatalog.k8sPrefix=foo-' \
      . | tee /dev/stderr |
      yq '.spec.template.spec.containers[0].command | any(contains("-k8s-service-prefix=\"foo-\""))' | tee /dev/stderr)
  [ "${actual}" = "true" ]
}

#--------------------------------------------------------------------
# k8sTag

@test "syncCatalog/Deployment: no k8sTag flag by default" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/sync-catalog-deployment.yaml  \
      --set 'syncCatalog.enabled=true' \
      . | tee /dev/stderr |
      yq '.spec.template.spec.containers[0].command | any(contains("-consul-k8s-tag"))' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "syncCatalog/Deployment: can specify k8sTag" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/sync-catalog-deployment.yaml  \
      --set 'syncCatalog.enabled=true' \
      --set 'syncCatalog.k8sTag=clusterB' \
      . | tee /dev/stderr |
      yq '.spec.template.spec.containers[0].command | any(contains("-consul-k8s-tag=clusterB"))' | tee /dev/stderr)
  [ "${actual}" = "true" ]
}

#--------------------------------------------------------------------
# serviceAccount

@test "syncCatalog/Deployment: serviceAccount set when sync enabled" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/sync-catalog-deployment.yaml  \
      --set 'syncCatalog.enabled=true' \
      . | tee /dev/stderr |
      yq '.spec.template.spec.serviceAccountName | contains("sync-catalog")' | tee /dev/stderr)
  [ "${actual}" = "true" ]
}

#--------------------------------------------------------------------
# nodePortSyncType

@test "syncCatalog/Deployment: nodePortSyncType defaults to ExternalFirst" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/sync-catalog-deployment.yaml  \
      --set 'syncCatalog.enabled=true' \
      . | tee /dev/stderr |
      yq '.spec.template.spec.containers[0].command | any(contains("-node-port-sync-type=ExternalFirst"))' | tee /dev/stderr)
  [ "${actual}" = "true" ]
}

@test "syncCatalog/Deployment: can set nodePortSyncType to InternalOnly" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/sync-catalog-deployment.yaml  \
      --set 'syncCatalog.enabled=true' \
      --set 'syncCatalog.nodePortSyncType=InternalOnly' \
      . | tee /dev/stderr |
      yq '.spec.template.spec.containers[0].command | any(contains("-node-port-sync-type=InternalOnly"))' | tee /dev/stderr)
  [ "${actual}" = "true" ]
}

@test "syncCatalog/Deployment: can set nodePortSyncType to ExternalOnly" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/sync-catalog-deployment.yaml  \
      --set 'syncCatalog.enabled=true' \
      --set 'syncCatalog.nodePortSyncType=ExternalOnly' \
      . | tee /dev/stderr |
      yq '.spec.template.spec.containers[0].command | any(contains("-node-port-sync-type=ExternalOnly"))' | tee /dev/stderr)
  [ "${actual}" = "true" ]
}
