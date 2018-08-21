# etcd Quickstart Guide [![Slack](https://slack.minio.io/slack?type=svg)](https://slack.minio.io)
etcd is a distributed reliable key-value store for the most critical data of a distributed system, this guide explains on how to configure etcd and what its used for in Minio STS.

## Get started
### 1. Prerequisites
- Install and run etcd.v3 [instance](https://coreos.com/etcd/docs/latest/op-guide/container.html).

### 2. Setup Minio with etcd
Minio server expects environment variable for ETCD endpoints as `MINIO_ETCD_ENDPOINTS`, this environment variable takes a comma separated list of etcd servers that you want to use as the Minio STS credential backend.
```
export MINIO_ETCD_ENDPOINTS=127.0.0.1:2379
minio server /mnt/data
```

Once you have ETCD successfully configured proceed to configuring [WSO2 Quickstart Guide](https://docs.minio.io/docs/wso2-quickstart-guide) and [OPA Quickstart Guide](https://docs.minio.io/docs/opa-quickstart-guide)

## Explore Further
- [Minio STS Quickstart Guide](https://docs.minio.io/docs/minio-sts-quickstart-guide)
- [The Minio documentation website](https://docs.minio.io)

