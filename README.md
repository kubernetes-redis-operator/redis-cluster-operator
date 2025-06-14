# Redis Cluster Operator

[![Operator Tests](https://github.com/kubernetes-redis-operator/redis-cluster-operator/actions/workflows/operator.yml/badge.svg)](https://github.com/kubernetes-redis-operator/redis-cluster-operator/actions/workflows/operator.yml)  [![Coverage Status](https://coveralls.io/repos/github/kubernetes-redis-operator/redis-cluster-operator/badge.svg?branch=main)](https://coveralls.io/github/kubernetes-redis-operator/redis-cluster-operator?branch=main)   [![Go Report Card](https://goreportcard.com/badge/github.com/kubernetes-redis-operator/redis-cluster-operator)](https://goreportcard.com/report/github.com/kubernetes-redis-operator/redis-cluster-operator)   [![CodeQL](https://github.com/kubernetes-redis-operator/redis-cluster-operator/actions/workflows/codeql.yml/badge.svg)](https://github.com/kubernetes-redis-operator/redis-cluster-operator/actions/workflows/codeql.yml)   [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

The Redis Cluster Operator runs Redis Clusters on Kubernetes.

We've found many operators which either use the redis-cli directly, which makes it hard to customise
behaviour, or do not support a full productionised suite of features.

The aim for this operator is to run productionised clusters with most necessary features, 
as well as providing additions such as RunBooks to help debug issues with Redis Clusters 
when running them with this Operator, and ready-made load tests to test your Redis Clusters with real traffic.

- [Redis Cluster Operator](#redis-cluster-operator)
  - [Features this operator supports](#features-this-operator-supports)
  - [Installing the Operator](#installing-the-operator)
    - [bundled cluster-wide](#bundled-cluster-wide)
    - [bundled namespaced](#bundled-namespaced)
    - [OLM bundle](#olm-bundle)
  - [Creating your first Redis Cluster](#creating-your-first-redis-cluster)

## Features this operator supports
- [x] Cluster Creation
- [ ] Cluster Management
- [x] Support for replicated clusters (Master-Replica splits)
- [ ] 0 Downtime scaling
- [ ] 0 Downtime upgrades
- [ ] Persistent clusters (Supported through Kubernetes PVC management)
- [ ] Backup & Restore capability for persistent clusters
- [ ] Documentation on observability for clusters
- [ ] Runbooks for common debugging issues and resolutions
- [ ] Ready-made k6s load tests to load Redis Clusters

## Installing the Operator

### bundled cluster-wide

The operator gets bundled for every release together with all of it's crds, rbac, and deployment.

The origin bundle works in cluster mode, and will manage all RedisClusters created in all namespaces. 

To install or upgrade the operator 
```shell
kubectl apply -f https://github.com/kubernetes-redis-operator/redis-cluster-operator/releases/latest/download/bundle.yml
```

This will install the Operator in a new namespace `redis-cluster-operator`. 

You can also [install the operator in a custom namespace](./docs/installing-in-a-custom-namespace.md).

### bundled namespaced

The operator currently works in cluster-wide mode, but the Operator will support namespaced mode in the future.

We know it's quite important for redundancy, reducing single-point of failures, 
as well as tenanted models, or excluding namespaces from the operator.

The Operator will support Namespaced mode in the future.

### OLM bundle

> OLM bundling support is a work in progress. 
> There are remnants of OLM due to the initial Operator SDK installation, 
> but we have not specifically tested and looked at it in depth.

## Creating your first Redis Cluster

To create your first Redis cluster, you'll need a CRD.

```yaml
apiVersion: rediscluster.kuro.io/v1alpha1
kind: RedisCluster
metadata:
  name: rediscluster-product-api
spec:
  masters: 3
  replicasPerMaster: 1
```

Once applied, the Operator will create all the necessary nodes, and set up the cluster ready for use.

Remember to check out the [documentation](./docs/home.md) page for more information.
