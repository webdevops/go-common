# Prometheus common library webdevops projects

## Collector

### Caching

Prometheus collector offers a caching mechanism for faster restart with different backends defined by the cache spec:

| Cache spec                                                                           | Description                                                          |
|--------------------------------------------------------------------------------------|----------------------------------------------------------------------|
| `file://path/to/cache/`                                                              | Use local filesystem to cache data (use PVC inside Kubernetes!)      |
| `azblob://{storageAccountName}.blob.core.windows.net/{containerName}/{optionalPath}` | Use Azure StorageAccount to save cache data (with optional sub path) |
| `k8scm://{namespace}/{configMapName}`                                                | Use Kubernetes ConfigMap to save cache data                          |
