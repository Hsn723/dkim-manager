# Upgrading dkim-manager

## v2 API introduction

### What changed

This release introduces a **v2 API** for `DKIMKey` resources with standard Kubernetes status conditions, enabling compatibility with GitOps tools like Flux CD that perform health checks on custom resources.

The v1 API is preserved and continues to work. A conversion webhook transparently translates between v1 and v2, so **no manual migration is required**.

**v1 status (unchanged):**
```yaml
status: ok
```

**v2 status (new):**
```yaml
status:
  observedGeneration: 1
  conditions:
    - type: Ready
      status: "True"
      reason: Succeeded
      message: "DKIM key created successfully"
      lastTransitionTime: "2026-01-01T00:00:00Z"
      observedGeneration: 1
```

### Upgrade procedure

Upgrade the Helm release as usual:

```bash
helm upgrade <release-name> dkim-manager/dkim-manager
```

Existing `DKIMKey` resources are automatically handled by the conversion webhook. No deletion, recreation, or manual intervention is needed.

### Verifying the upgrade

After upgrading, existing resources should continue to report their status:

```bash
# Via v2 API (new conditions-based status)
kubectl get dkimkey -n <namespace>

# Via v1 API (original string status, still works)
kubectl get dkimkey.v1.dkim-manager.atelierhsn.com -n <namespace>
```

### Flux CD users

After upgrading, you can use `wait: true` on Kustomizations that include DKIMKey resources. Flux will properly health-check them via `.status.observedGeneration` and `.status.conditions`.
