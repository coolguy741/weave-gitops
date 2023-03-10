## gitops add

Add a new Weave GitOps resource

### Examples

```

# Add a new cluster using a CAPI template
gitops add cluster
```

### Options

```
  -h, --help   help for add
```

### Options inherited from parent commands

```
  -e, --endpoint string            The Weave GitOps Enterprise HTTP API endpoint
      --insecure-skip-tls-verify   If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure
      --namespace string           The namespace scope for this operation (default "flux-system")
  -v, --verbose                    Enable verbose output
```

### SEE ALSO

* [gitops](gitops.md)	 - Weave GitOps
* [gitops add cluster](gitops_add_cluster.md)	 - Add a new cluster using a CAPI template
* [gitops add profile](gitops_add_profile.md)	 - Add a profile to a cluster

###### Auto generated by spf13/cobra on 12-Apr-2022
