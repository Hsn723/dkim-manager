# Upgrading dkim-manager

## Upgrading to v1.3.0 (or later)

### Breaking Change: Status Format

Version 1.3.0 changes the `DKIMKey` status from a simple string to a standard Kubernetes status object with `observedGeneration` and `conditions`. This enables compatibility with GitOps tools like Flux CD that perform health checks on custom resources.

**Before (v1.2.x and earlier):**
```yaml
status: ok
```

**After (v1.3.0+):**
```yaml
status:
  observedGeneration: 1
  conditions:
    - type: Ready
      status: "True"
      reason: Succeeded
      message: "DKIM key created successfully"
      lastTransitionTime: "2024-01-01T00:00:00Z"
      observedGeneration: 1
```

### Migration Procedure

Existing `DKIMKey` resources with the old status format are incompatible with the new controller. You must delete and recreate them.

**Important:** The DKIM private keys stored in Secrets and DNS records created via DNSEndpoint are managed by the DKIMKey controller. Deleting DKIMKey resources will also delete these associated resources. If you need to preserve existing keys, back them up first.

#### Step 1: Backup existing secrets (optional)

```bash
# List existing DKIM secrets
kubectl get secrets -n <namespace> -l app.kubernetes.io/managed-by=dkim-manager

# Backup if needed
kubectl get secret <secret-name> -n <namespace> -o yaml > dkim-secret-backup.yaml
```

#### Step 2: Delete the validating webhook

The webhook will reject operations on resources with the old status format:

```bash
kubectl delete validatingwebhookconfiguration <release-name>-dkim-manager-validating-webhook-configuration
```

#### Step 3: Remove finalizers from existing DKIMKey resources

```bash
# List all DKIMKey resources
kubectl get dkimkey --all-namespaces

# Remove finalizers (repeat for each resource)
kubectl patch dkimkey <name> -n <namespace> \
  -p '{"metadata":{"finalizers":null}}' --type=merge
```

#### Step 4: Delete existing DKIMKey resources

```bash
kubectl delete dkimkey --all --all-namespaces
```

#### Step 5: Upgrade the Helm release

```bash
helm upgrade <release-name> dkim-manager/dkim-manager --version 1.3.0
```

#### Step 6: Recreate DKIMKey resources

Apply your DKIMKey manifests again. The controller will generate new keys and create the associated Secret and DNSEndpoint resources.

```bash
kubectl apply -f your-dkimkey-manifests.yaml
```

### Verifying the Upgrade

After recreating DKIMKey resources, verify they have the new status format:

```bash
kubectl get dkimkey -n <namespace> -o yaml
```

You should see:
```yaml
status:
  observedGeneration: 1
  conditions:
    - type: Ready
      status: "True"
      ...
```

### Flux CD Users

After upgrading, you can use `wait: true` on Kustomizations that include DKIMKey resources. Flux will now properly health-check them by looking at `.status.observedGeneration` and `.status.conditions`.
