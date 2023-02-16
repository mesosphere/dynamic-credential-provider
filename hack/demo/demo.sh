#!/bin/bash
set -eou pipefail
IFS=$'\n\t'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR

pushd "${SCRIPT_DIR}/../.."

readonly DEMODATA_DIR="demodata"
rm -rf "${DEMODATA_DIR}" && mkdir -p "${DEMODATA_DIR}"

REGISTRY_PORT=5000
# Use a domain so it can be access on a Mac
REGISTRY_ADDRESS=registry
REGISTRY_CERTS_DIR="$(realpath "${DEMODATA_DIR}/certs")"
REGISTRY_AUTH_DIR="$(realpath "${DEMODATA_DIR}/auth")"
REGISTRY_USERNAME=testuser
REGISTRY_PASSWORD=testpassword

# Create certs for the registry
rm -rf "${REGISTRY_CERTS_DIR}"
mkdir -p "${REGISTRY_CERTS_DIR}"

OPENSSL_BIN="${OPENSSL_BIN:=openssl}"

${OPENSSL_BIN} req -x509 -newkey rsa:4096 -sha256 -days 3650 -nodes \
  -keyout "${REGISTRY_CERTS_DIR}/ca.key" -out "${REGISTRY_CERTS_DIR}/ca.crt" -subj "/CN=${REGISTRY_ADDRESS}" \
  -addext "subjectAltName=DNS:${REGISTRY_ADDRESS}"

rm -rf "${REGISTRY_AUTH_DIR}"
mkdir -p "${REGISTRY_AUTH_DIR}"
docker container run --rm \
  --entrypoint htpasswd \
  httpd:2 -Bbn "${REGISTRY_USERNAME}" "${REGISTRY_PASSWORD}" >"${REGISTRY_AUTH_DIR}/htpasswd"

docker container rm -fv registry || true
docker container run --rm -d \
  --name registry \
  --network kind \
  -v "${REGISTRY_CERTS_DIR}":/certs \
  -e REGISTRY_HTTP_TLS_CERTIFICATE=/certs/ca.crt \
  -e REGISTRY_HTTP_TLS_KEY=/certs/ca.key \
  -v "${REGISTRY_AUTH_DIR}":/auth \
  -e "REGISTRY_AUTH=htpasswd" \
  -e "REGISTRY_AUTH_HTPASSWD_REALM=Registry Realm" \
  -e REGISTRY_AUTH_HTPASSWD_PATH=/auth/htpasswd \
  -p "${REGISTRY_PORT}":"${REGISTRY_PORT}" \
  registry:2

docker image pull nginx:latest
docker image tag docker.io/library/nginx:latest "127.0.0.1:${REGISTRY_PORT}/library/nginx:latest"

echo "${REGISTRY_PASSWORD}" | docker login -u "${REGISTRY_USERNAME}" --password-stdin "127.0.0.1:${REGISTRY_PORT}"
docker image push "127.0.0.1:${REGISTRY_PORT}/library/nginx:latest"

# Retag and push with a tag that doesn't exist in docker.io to test Containerd mirror config
docker image tag docker.io/library/nginx:latest "127.0.0.1:${REGISTRY_PORT}/library/nginx:$(whoami)"
docker image push "127.0.0.1:${REGISTRY_PORT}/library/nginx:$(whoami)"

docker build -f "hack/demo/Dockerfile.demo-node" -t mesosphere/kind-node-demo-credential-provider:v1.25.4 .

export KUBECONFIG="${DEMODATA_DIR}"/bootstrap-kubeconfig

kind delete clusters image-credential-provider-bootstrap || true
kind create cluster --name image-credential-provider-bootstrap --image mesosphere/kind-node:v1.25.4 --config <(
  cat <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."docker.io"]
    endpoint = ["https://mirror.gcr.io"]
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.configs."registry-1.docker.io".auth]
    username = "${DOCKER_HUB_USERNAME-}"
    password = "${DOCKER_HUB_PASSWORD-}"
nodes:
- role: control-plane
  extraMounts:
    - hostPath: /var/run/docker.sock
      containerPath: /var/run/docker.sock
EOF
)

# Enable the experimental Cluster topology and cluster resource set features.
export CLUSTER_TOPOLOGY=true EXP_CLUSTER_RESOURCE_SET=true

# Initialize the management cluster
clusterctl init --infrastructure docker --wait-providers

kubectl apply -f "hack/demo/docker-clusterclass.yaml"

kubectl create configmap tigera-operator \
  --from-file=tigera-operator.json=<(curl -fsSL "https://docs.projectcalico.org/archive/v3.24/manifests/tigera-operator.yaml" |
    gojq --yaml-input \
      --slurp \
      --indent=0 \
      '[ .[] | select( . != null ) | (select(.kind == "Namespace").metadata.labels={"pod-security.kubernetes.io/enforce": "privileged"}) ]')
kubectl create configmap calico-installation --from-file=custom-resources.yaml=<(
  cat <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: calico-system
  labels:
    pod-security.kubernetes.io/enforce: privileged
---
apiVersion: operator.tigera.io/v1
kind: Installation
metadata:
  name: default
spec:
  cni:
    type: Calico
  calicoNetwork:
    ipPools:
    - blockSize: 26
      cidr: 192.168.0.0/16
      encapsulation: VXLANCrossSubnet
      natOutgoing: Enabled
      nodeSelector: all()
EOF
)
kubectl apply -f <(
  cat <<EOF
apiVersion: addons.cluster.x-k8s.io/v1beta1
kind: ClusterResourceSet
metadata:
  name: calico-cni
spec:
  clusterSelector:
    matchLabels:
      cni: calico
  resources:
    - kind: ConfigMap
      name: tigera-operator
    - kind: ConfigMap
      name: calico-installation
EOF
)

kubectl apply -f <(
  cat <<EOF
apiVersion: cluster.x-k8s.io/v1beta1
kind: Cluster
metadata:
  name: "shim-demo"
  labels:
    cni: calico
spec:
  clusterNetwork:
    services:
      cidrBlocks: ["10.128.0.0/12"]
    pods:
      cidrBlocks: ["192.168.0.0/16"]
    serviceDomain: "cluster.local"
  topology:
    class: shim-demo
    controlPlane:
      metadata: {}
      replicas: 1
    variables:
    - name: registryMirrors
      value:
        - registryHost: "*"
          mirrorEndpoints:
            - "https://${REGISTRY_ADDRESS}:${REGISTRY_PORT}"
        - registryHost: "docker.io"
          mirrorEndpoints:
            - "https://${REGISTRY_ADDRESS}:${REGISTRY_PORT}"
        - registryHost: "k8s.gcr.io"
          mirrorEndpoints:
            - "https://${REGISTRY_ADDRESS}:${REGISTRY_PORT}"
    - name: registryCredentials
      value:
        - registryHost: "${REGISTRY_ADDRESS}:${REGISTRY_PORT}"
          staticCredentials:
            username: "${REGISTRY_USERNAME}"
            password: "${REGISTRY_PASSWORD}"
        - registryHost: "docker.io"
          staticCredentials:
            username: "${DOCKER_HUB_USERNAME-}"
            password: "${DOCKER_HUB_PASSWORD-}"
    - name: registryCredentialStrategy
      value: MirrorCredentialsFirst
    - name: kindNodeImageRepository
      value: "mesosphere/kind-node-demo-credential-provider"
    - name: imageRepository
      value: ""
    - name: etcdImageTag
      value: "3.5.5-0"
    - name: coreDNSImageTag
      value: ""
    - name: podSecurityStandard
      value:
        enabled: true
        enforce: "baseline"
        audit: "restricted"
        warn: "restricted"
    version: 1.25.4
    workers:
      machineDeployments:
      - class: default-worker
        name: md-0
        replicas: 1
EOF
)

kubectl wait --for=condition=ControlPlaneReady clusters/shim-demo --timeout=2m

clusterctl get kubeconfig shim-demo >"${DEMODATA_DIR}/cluster-kubeconfig"

export KUBECONFIG="${DEMODATA_DIR}/cluster-kubeconfig"

until kubectl get serviceaccount default &>/dev/null; do
  echo "Waiting until default serviceaccount exists..."
  sleep 1
done

# Create a Pod with an image that is pulled from the registry
kubectl run nginx-latest --image=registry:5000/library/nginx:latest --image-pull-policy=Always
# Create a Pod with an image tag that does not exist in docker.io and exercise Containerd mirror
kubectl run nginx-mirror --image="docker.io/library/nginx:$(whoami)" --image-pull-policy=Always
# Create a Pod with an image tag that does not exist in registry:5000, but exists in docker.io
kubectl run nginx-stable --image=docker.io/library/nginx:stable --image-pull-policy=Always

kubectl wait --for=condition=Ready pods --all --timeout=5m
