# harpoon

[![Docker](https://github.com/adrianchifor/harpoon/workflows/Publish%20Docker/badge.svg)](https://github.com/adrianchifor/harpoon/actions?query=workflow%3A%22Publish+Docker%22) [![Go Report Card](https://goreportcard.com/badge/github.com/adrianchifor/harpoon)](https://goreportcard.com/report/github.com/adrianchifor/harpoon)

Pre-pull Docker images on Kubernetes nodes to speed up containers bootstrap and autoscaling.

It can automatically discover the most popular images from Pods in the cluster or just pull manually specified ones.

Supports both Docker and cri-o.

## Setup

### Automatic

```
$ kubectl apply -f deploy/daemonset-auto.yaml

serviceaccount/harpoon created
clusterrole.rbac.authorization.k8s.io/harpoon created
clusterrolebinding.rbac.authorization.k8s.io/harpoon created
daemonset.apps/harpoon created
```

This will run harpoon as the init container in a DaemonSet and then run pause as the main container, so we keep the pod up and we don't leave the host Docker/cri-o socket exposed.

It will also setup RBAC so harpoon can get/list Pods in the cluster.

By default it gets the Docker images from Pods in the same namespace and pulls them. You can define `NAMESPACES` env var and tell harpoon to scan other namespaces:

```
initContainers:
  - name: harpoon
    image: adrianchifor/harpoon:latest
    env:
      - name: NAMESPACES
        value: "ns1,ns2"
```

Or all namespaces:

```
env:
  - name: NAMESPACES
    value: "*"
```

You can also specify a limit of how many images to pull (sorted by most popular first):

```
env:
  - name: LIMIT
    value: "10"
```

If you want to ignore some images, specify their prefixes as a comma-separated list:

```
env:
  - name: IGNORE
    value: "quay.io/openshift-release-dev/,k8s.gcr.io/"
```

To use a private registry with cri-o, specify the registry name and auth (base64 of username:password).

Any image that contains the `PRIVATE_REGISTRY` will use `crictl pull --auth $PRIVATE_REGISTRY_AUTH <image>`.

```
env:
  - name: PRIVATE_REGISTRY
    value: registry.company.com
  - name: PRIVATE_REGISTRY_AUTH
    valueFrom:
      secretKeyRef:
        name: registry-auth
        key: auth
```

### Manual

If you want to pre-pull specific Docker images and skip the Pod checks, you can list your images in `/config/images` in the container. See [daemonset-manual.yaml](./deploy/daemonset-manual.yaml) example.

```
$ kubectl apply -f deploy/daemonset-manual.yaml

configmap/harpoon-images created
daemonset.apps/harpoon created
```
