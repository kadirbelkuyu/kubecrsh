# Kubernetes Manifests

Bu dizin kubecrsh'nin Kubernetes cluster'a deploy edilmesi için gerekli tüm manifest dosyalarını içerir.

## Dosyalar

- **rbac.yaml** - ServiceAccount, ClusterRole, ClusterRoleBinding
- **configmap.yaml** - Uygulama konfigürasyonu
- **deployment.yaml** - kubecrsh daemon deployment
- **service.yaml** - Metrics endpoint service
- **secret.yaml.example** - Secret template (webhook URL'leri için)

## Kullanım

### Tüm manifests'leri deploy et

```bash
kubectl apply -f manifests/
```

### Secret oluştur (isteğe bağlı)

```bash
kubectl create secret generic kubecrsh-secrets \
  --from-literal=slack-webhook='https://hooks.slack.com/services/xxx' \
  -n kubecrsh
```

### Verify

```bash
kubectl get all -n kubecrsh
```

### Uninstall

```bash
kubectl delete -f manifests/
```
