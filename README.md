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
