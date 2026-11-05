# OpenTofu OCI Mirror

Each release publishes its platform packages to GHCR as an OCI artifact in the
layout OpenTofu's `oci_mirror` installation method reads. The OCI tag is the
release version without its leading `v`; for example `v1.2.1` is `1.2.1`.

The mirror contains the same zip files as the GitHub release, so the provider
source address in HCL does not change. OpenTofu resolves the provider from GHCR
instead of the origin registry.

## Configure OpenTofu

Exclude the origin registry explicitly; without the matching `direct` exclude,
OpenTofu may still install from the origin.

```hcl
# ~/.tofurc
provider_installation {
  oci_mirror {
    repository_template = "ghcr.io/arsolitt/terraform-provider-garagehq"
    include             = ["registry.opentofu.org/arsolitt/garagehq"]
  }
  direct {
    exclude             = ["registry.opentofu.org/arsolitt/garagehq"]
  }
}
```

Then `tofu init` resolves `arsolitt/garagehq` from the mirror:

```hcl
terraform {
  required_providers {
    garage = {
      source  = "arsolitt/garagehq"
      version = "1.2.1"
    }
  }
}
```

The `include` address must match the source address written in HCL. If you write
the host explicitly, use the same host in `include`.

## Requirements and limitations

- OpenTofu >= 1.10 for `oci_mirror`.
- OpenTofu >= 1.13 for GHCR. OpenTofu 1.10 through 1.12 ping `GET /v2/`; GHCR
  answers with a scope that the anonymous token request cannot use, and
  `tofu init` fails with `failed to contact OCI registry`. OpenTofu 1.13 removed
  that ping. On 1.10 through 1.12, configure `oci_credentials "ghcr.io"` with a
  GitHub token.
- A package published from a public repository is public. A private package
  needs `oci_credentials`.
- Prereleases are selected only by a version constraint that names the
  prerelease version.

## Verify a published release

```bash
oras manifest fetch ghcr.io/arsolitt/terraform-provider-garagehq:1.2.1
```

The index has artifact type `application/vnd.opentofu.provider`, one
`application/vnd.opentofu.provider-target` manifest per platform, and each
manifest's single `archive/zip` layer is the matching release zip. The layer
digest equals the zip's SHA-256.
