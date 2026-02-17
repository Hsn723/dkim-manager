# dkim-manager Helm Chart

## Quick start

### Add the Helm Repository
```sh
helm repo add dkim-manager https://hsn723.github.io/dkim-manager
helm repo update
```

### Install cert-manager
```sh
curl -sLf https://github.com/jetstack/cert-manager/releases/latest/download/cert-manager.yaml | kubectl apply -f -
```

### Install external-dns CRD
```sh
helm show crds external-dns/external-dns --version "$(yq .dependencies[0].version Chart.yaml)" | kubectl apply -f -
```

### Install the Chart

Installing the chart with default settings (standalone):

```sh
helm install --create-namespace --namespace dkim-manager dkim-manager dkim-manager/dkim-manager
```

Specify parameters using `--set key=value[,key=value]` arguments to `helm install`, or provide your own `values.yaml`:

```sh
helm install --create-namespace --namespace dkim-manager dkim-manager -f values.yaml dkim-manager/dkim-manager
```

## Values
| Key | Type | Default | Description |
|-----|------|---------|-------------|
| image.repository | string | `"ghcr.io/hsn723/dkim-manager"` | Image repository to use |
| image.tag | string | `{{ .Chart.AppVersion }}` | Image tag to use |
| image.pullPolicy | string | "Always" | Image pullPolicy |
| controller.replicas | int | `2` | Number of controller Pod replicas |
| controller.resources | object | `{"requests":{"cpu":100m,"memory":"20Mi"}}` | Resources requested for controller Pod |
| controller.terminationGracePeriodSeconds | int | `10` | terminationGracePeriodSeconds for the controller Pod |
| controller.extraArgs | list | `["--leader-elect"]` | Additional arguments for the controller |
| namespaced | bool | `false` | Only look for DKIMKeys in the same namespace |
| namespace | string | `""` | Specify namespace in which to look for DKIMKeys |
| external-dns.enabled | bool | `false` | Also deploy the `external-dns` chart bundled for convenience |
| external-dns | object | | Custom values for the external-dns chart |

The `external-dns` helm chart is included for convenience. If you use it, you must provide some extra values in `values.yaml` to suit your environment. For instance, specifying environments `external-dns.env` for the DNS provider in use, changing the default provider in `external-dns.provider` if needed, adding `--namespace=YOUR_NAMESPACE` to `external-dns.extraArgs` to run `external-dns` only against the namespace `dkim-manager` is deployed to, etc.

## Generate Manifests
```sh
helm template --namespace dkim-manager dkim-manager [-f values.yaml] dkim-manager/dkim-manager
```
