## Notes 17/08/23
- KV stores should not necessarily be reused between bank-vaults and secret-sync
- Check how external KMS implemented their providers to get idea about how to move forward
- Mapping function should be handled by provider

- Rules API should allow options on an individual secret level (e.g. merge, transform, concat...) via e.g. templating
- Enable path for multi-destination source/dest if optimal (changes to rules API will have to change)
- Have one "global" synchronization configuration file
- Load configurations for sources/dests from e.g. Kubernetes CRs (check how external secrets k8s works to enable support for more KV stores)
- Documentation (stress the importance of early version)

Priority:
- Ability to have rules api for secrets on individual level (can be added later)
- Add more providers (e.g. AWS, k8s)
- Secret store (actually: set store) != KV store
