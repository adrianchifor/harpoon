# harpoon

[![Docker](https://github.com/adrianchifor/harpoon/workflows/Publish%20Docker/badge.svg)](https://github.com/adrianchifor/harpoon/actions?query=workflow%3A%22Publish+Docker%22) [![Go Report Card](https://goreportcard.com/badge/github.com/adrianchifor/harpoon)](https://goreportcard.com/report/github.com/adrianchifor/harpoon)

Pre-pull Docker images on Kubernetes nodes to speed up containers bootstrap and autoscaling. It can automatically discover images from Deployments or just pull manually specified ones. Supports both Docker and cri-o.

## Setup

### Automatic

```
$ kubectl apply -f deploy/daemonset-auto.yaml

serviceaccount/harpoon created
clusterrole.rbac.authorization.k8s.io/harpoon created
clusterrolebinding.rbac.authorization.k8s.io/harpoon created
daemonset.apps/harpoon created
```

This will run harpoon as the init container in a DaemonSet and then run pause as the main container, so we keep the pod up and we don't leave the host Docker/cri-o socket exposed. It will also setup RBAC so harpoon can get/list Deployments in the cluster.

By default it gets the Docker images from Deployments in the same namespace and pulls them. You can define `NAMESPACES` env var and tell harpoon to scan other namespaces:

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

**Why only check Deployment Docker images?**

Looking at all [Kubernetes workload controllers](https://kubernetes.io/docs/concepts/workloads/controllers/):
- Job/CronJob generally doesn't need quick autoscaling
- StatefulSet generally has a set number of pods
- ReplicationController is already managed by Deployment
- DaemonSet will run first on new nodes so it's likely the Docker image is already pulled when harpoon runs

If you think harpoon should check more workload controllers, please open a GitHub issue and we can discuss.

### Manual

If you want to pre-pull specific Docker images and skip the Deployment checks, you can list your images in `/config/images` in the container. See [daemonset-manual.yaml](./deploy/daemonset-manual.yaml) example.

```
$ kubectl apply -f deploy/daemonset-manual.yaml

configmap/harpoon-images created
daemonset.apps/harpoon created
```
