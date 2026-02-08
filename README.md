[![GitHub release](https://img.shields.io/github/release/hsn723/dkim-manager.svg?sort=semver&maxAge=60)](https://github.com/hsn723/dkim-manager/releases)
[![Helm release](https://img.shields.io/badge/dynamic/yaml.svg?label=chart&url=https://hsn723.github.io/dkim-manager/index.yaml&query=$.entries[%22dkim-manager%22][0].version&colorB=orange&logo=helm)](https://github.com/hsn723/dkim-manager/releases)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/dkim-manager)](https://artifacthub.io/packages/search?repo=dkim-manager)
[![main](https://github.com/Hsn723/dkim-manager/actions/workflows/main.yml/badge.svg?branch=master)](https://github.com/Hsn723/dkim-manager/actions/workflows/main.yml)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/hsn723/dkim-manager?tab=overview)](https://pkg.go.dev/github.com/hsn723/dkim-manager?tab=overview)
[![Go Report Card](https://goreportcard.com/badge/github.com/hsn723/dkim-manager)](https://goreportcard.com/report/github.com/hsn723/dkim-manager)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/hsn723/dkim-manager)

# dkim-manager
`dkim-manager` is a Kubernetes controller for creating DKIM keys.

## Motivation
When sending mail from inside a Kubernetes cluster, you might want to sign outgoing emails with a DKIM signature. Setting up DKIM involves placing a private key for the DKIM signer to consume, and registering a DNS record containing the public key. Doing so manually can be a chore, and in some environments it is not acceptable to create private keys locally. `dkim-manager` aims to facilitate this process.

## Features
`dkim-manager` is used in combination with [external-dns](https://github.com/kubernetes-sigs/external-dns). When a DKIM key is requested via the `DKIMKey` custom resource, `dkim-manager` creates a key pair, and creates two resources:

- a `Secret` containing the private key, that the mailer pod can mount and consume
- a `DNSEndpoint` containing the public key and other necessary information for `external-dns` to create the DNS record
- RSA (1024-bit, 2048-bit, 4096-bit) and ed25519 keys are supported
    - 2048-bit RSA is selected as a sensible default

It is recommended to create a delegated subdomain for the sole purpose of storing DKIM records (eg: `dkim.example.com`) and grant `external-dns` only permission on that subdomain. See [this blog](https://atelierhsn.com/2022/01/cert-manager-done-right/) for more details why.

## Upgrading

See [UPGRADING.md](UPGRADING.md) for upgrade instructions, especially when upgrading to v1.3.0+ which includes a breaking change to the status format.

## Installation
`dkim-manager` requires `cert-manager` and `external-dns` to be installed first. The [helm installation instructions](charts/dkim-manager/README.md) are a good place to get started. If installing `external-dns` separately, not that the following arguments should be set for `dkim-manager` to be able to register TXT records:

```yaml
- --source=crd
- --crd-source-apiversion=externaldns.k8s.io/v1alpha1
- --crd-source-kind=DNSEndpoint
- --managed-record-types=TXT
- --txtPrefix= #some non-empty string
```

Additionally, it is recommended to set `--domainFilter` to restrict the scope of operation of `external-dns` to the domain for which you want to create DKIM keys, and to set `--namespace=YOUR_NAMESPACE` so that `external-dns` only looks at resources inside your namespace. Doing so allows you to use `external-dns` for the sole purpose of registering DKIM TXT records.

## Usage
DKIM keys can be requested by creating a `DKIMKey` resource.

```yaml
apiVersion: dkim-manager.atelierhsn.com/v1
kind: DKIMKey
metadata:
    name: selector1-example-com
    namespace: example
spec:
    secretName: selector1-example-com
    selector: selector1
    domain: dkim.example.com
```

This will create the following resources:

```yaml
# Secret
apiVersion: v1
kind: Secret
metadata:
    name: selector1-example-com
data:
    dkim.example.com.selector1.key: |
        "..."
---
### DNSEndpoint
apiVersion: externaldns.k8s.io/v1alpha1
kind: DNSEndpoint
metadata:
    name: selector1-example-com
spec:
    endpoints:
    - dnsName: selector1._domainkey.dkim.example.com
      recordTTL: 86400
      recordType: TXT
      targets:
      - "v=DKIM1; h=sha256; k=rsa; p=...."
```

## Future Considerations
Currently, DKIM private keys are stored as a `Secret` resource. While ubiquitous, this makes the keys visible to any priviledged users inside the cluster. In a future release support for writing private keys to [HashiCorp Vault](https://www.vaultproject.io/) may be considered.
