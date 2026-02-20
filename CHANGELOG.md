# Changelog

## [1.0.0](https://github.com/Arsolitt/terraform-provider-garagehq/compare/v0.0.16...v1.0.0) (2026-02-20)


### ⚠ BREAKING CHANGES

* rename provider to arsolitt/garagehq
* **bucket:** Resources will now be actually deleted from Garage
* **client:** Requires Garage v2.x server
* Requires Garage v2.x server

### Features

* add garage_admin_token resource ([b3f049f](https://github.com/Arsolitt/terraform-provider-garagehq/commit/b3f049fd65461db5327b1d596548fe4db64404df))
* add garage_cluster_layout resource ([23b5e3d](https://github.com/Arsolitt/terraform-provider-garagehq/commit/23b5e3d3fcdd648d58d5a866687f424aebff2231))
* **bucket:** implement delete with api v2 ([ffc90c2](https://github.com/Arsolitt/terraform-provider-garagehq/commit/ffc90c25204eb9df3ac4806cc18c19b3a2f1093d))
* initial commit of terraform-provider-garage ([7ddae69](https://github.com/Arsolitt/terraform-provider-garagehq/commit/7ddae692ba842e6780cb875a5dcdc2e9a21a06c7))
* **provider:** register garage_admin_token and garage_cluster_layout resources ([2ca3477](https://github.com/Arsolitt/terraform-provider-garagehq/commit/2ca3477a826778da861162e4066ded7cf8c7a3a6))
* update release workflow to build and upload binaries for all pl… ([1a3766d](https://github.com/Arsolitt/terraform-provider-garagehq/commit/1a3766dc79e4fbbbecdc5b65cace93d36705a6d9))
* update release workflow to build and upload binaries for all platforms ([f00b33e](https://github.com/Arsolitt/terraform-provider-garagehq/commit/f00b33e4bc7f1abd8b422905b3754f638d3359b4))


### Bug Fixes

* add --pinentry-mode loopback and GPG_TTY for GPG signing in CI ([f79a23d](https://github.com/Arsolitt/terraform-provider-garagehq/commit/f79a23d8c02b799de7d08e4b22530d1d4ade9cc2))
* add binary format archives to match Grafana provider pattern ([2e48006](https://github.com/Arsolitt/terraform-provider-garagehq/commit/2e4800679a613526c66b76f9cabdd530efa569c9))
* address golangci-lint errors - check error returns and fix deprecated API ([3ac1e29](https://github.com/Arsolitt/terraform-provider-garagehq/commit/3ac1e298e4b1ded30b19e9f0d8aa39065e83a780))
* address remaining golangci-lint errors in resource files ([8cb08b7](https://github.com/Arsolitt/terraform-provider-garagehq/commit/8cb08b77e5899f1adb5172876c9577242343b2be))
* **ci:** prevent dev-build from running on PRs and fix YAML syntax ([2626716](https://github.com/Arsolitt/terraform-provider-garagehq/commit/26267168ac17164279c452b4c21121a256ffb028))
* configure GPG for non-interactive signing in CI ([4ba876f](https://github.com/Arsolitt/terraform-provider-garagehq/commit/4ba876f8fa423d42a352a31e6aa4b7775e4a9690))
* correct protocol version in manifest to 5.0 only ([5b362a2](https://github.com/Arsolitt/terraform-provider-garagehq/commit/5b362a2f4362905cccf9bcc4212a70707d607a09))
* lint ([6a6ec1b](https://github.com/Arsolitt/terraform-provider-garagehq/commit/6a6ec1b53b1de72b22505fb605255866532c589c))
* match Grafana provider release pattern exactly ([3b19fc0](https://github.com/Arsolitt/terraform-provider-garagehq/commit/3b19fc030bfbf2fa7038313fb0a165f5e18ace24))
* match Grafana provider release pattern exactly ([8a19e83](https://github.com/Arsolitt/terraform-provider-garagehq/commit/8a19e838c93680f79c4dcde52aae4e75c5053140))
* pass GPG passphrase to GoReleaser signing ([40eba09](https://github.com/Arsolitt/terraform-provider-garagehq/commit/40eba09e5a78dc0dd4f282bcc14ce99f68c9cb5d))
* provide GPG passphrase to GoReleaser for signing ([54dbc5b](https://github.com/Arsolitt/terraform-provider-garagehq/commit/54dbc5b484f3998830e81a92624b0a6bf4181c62))
* remove confusing fallback token from release-please workflow ([b96d589](https://github.com/Arsolitt/terraform-provider-garagehq/commit/b96d5892ee9c49ea694f59a01034968be4735c31))
* resolve YAML linting issues ([06846fc](https://github.com/Arsolitt/terraform-provider-garagehq/commit/06846fc86ac8ec915b7b3acc384f5b5919b6b962))
* resolve YAML linting issues ([7cf1417](https://github.com/Arsolitt/terraform-provider-garagehq/commit/7cf1417fa8d775aa26bc8301e3b9f361136c40bd))
* test release ([237a153](https://github.com/Arsolitt/terraform-provider-garagehq/commit/237a15373b880a4ab8a513ec460c16cffab35a88))
* test release ([ec3d3bb](https://github.com/Arsolitt/terraform-provider-garagehq/commit/ec3d3bb324814236aaf2c49a08a0345aa4ad832a))
* test release please ([46ed768](https://github.com/Arsolitt/terraform-provider-garagehq/commit/46ed768bd440ce25f40d4d899fa85101a1e00eda))
* update GoReleaser config to version 2 and fix GPG signing ([f7dc6f7](https://github.com/Arsolitt/terraform-provider-garagehq/commit/f7dc6f70105e036bbd76566447f9fb85e0f5ba20))
* update workflows to handle branch protection and missing token ([cccfea7](https://github.com/Arsolitt/terraform-provider-garagehq/commit/cccfea778993c32d346e0f5dc8b7ef255437f67c))
* use bash wrapper script for GPG signing with passphrase ([6c2cbc4](https://github.com/Arsolitt/terraform-provider-garagehq/commit/6c2cbc4bed3154c27a2194be7ab0f4c5c526e00f))
* use GITHUB_TOKEN as fallback for release-please ([c5d9fc1](https://github.com/Arsolitt/terraform-provider-garagehq/commit/c5d9fc1552aae32f114b72587d64c3b9efce60e1))


### Code Refactoring

* **client:** migrate to garage admin api v2 ([39b0d26](https://github.com/Arsolitt/terraform-provider-garagehq/commit/39b0d260e018007be0fee38c6987a23ad04a74ef))
* rename provider to arsolitt/garagehq ([6fc68f3](https://github.com/Arsolitt/terraform-provider-garagehq/commit/6fc68f319a4e44d6b73daea00d71b16237d7615d))


### Build System

* update garage-admin-sdk to v2 ([d9d1493](https://github.com/Arsolitt/terraform-provider-garagehq/commit/d9d14932dd1ee20829aa0a0f0f73634eb76c0015))

## [0.0.16](https://github.com/d0ugal/terraform-provider-garage/compare/v0.0.15...v0.0.16) (2025-11-20)


### Bug Fixes

* correct protocol version in manifest to 5.0 only ([5b362a2](https://github.com/d0ugal/terraform-provider-garage/commit/5b362a2f4362905cccf9bcc4212a70707d607a09))

## [0.0.15](https://github.com/d0ugal/terraform-provider-garage/compare/v0.0.14...v0.0.15) (2025-11-20)


### Bug Fixes

* pass GPG passphrase to GoReleaser signing ([40eba09](https://github.com/d0ugal/terraform-provider-garage/commit/40eba09e5a78dc0dd4f282bcc14ce99f68c9cb5d))

## [0.0.14](https://github.com/d0ugal/terraform-provider-garage/compare/v0.0.13...v0.0.14) (2025-11-20)


### Bug Fixes

* add --pinentry-mode loopback and GPG_TTY for GPG signing in CI ([f79a23d](https://github.com/d0ugal/terraform-provider-garage/commit/f79a23d8c02b799de7d08e4b22530d1d4ade9cc2))

## [0.0.13](https://github.com/d0ugal/terraform-provider-garage/compare/v0.0.12...v0.0.13) (2025-11-20)


### Bug Fixes

* match Grafana provider release pattern exactly ([3b19fc0](https://github.com/d0ugal/terraform-provider-garage/commit/3b19fc030bfbf2fa7038313fb0a165f5e18ace24))
* match Grafana provider release pattern exactly ([8a19e83](https://github.com/d0ugal/terraform-provider-garage/commit/8a19e838c93680f79c4dcde52aae4e75c5053140))

## [0.0.12](https://github.com/d0ugal/terraform-provider-garage/compare/v0.0.11...v0.0.12) (2025-11-20)


### Bug Fixes

* add binary format archives to match Grafana provider pattern ([2e48006](https://github.com/d0ugal/terraform-provider-garage/commit/2e4800679a613526c66b76f9cabdd530efa569c9))

## [0.0.11](https://github.com/d0ugal/terraform-provider-garage/compare/v0.0.10...v0.0.11) (2025-11-20)


### Bug Fixes

* **ci:** prevent dev-build from running on PRs and fix YAML syntax ([2626716](https://github.com/d0ugal/terraform-provider-garage/commit/26267168ac17164279c452b4c21121a256ffb028))

## [0.0.10](https://github.com/d0ugal/terraform-provider-garage/compare/v0.0.9...v0.0.10) (2025-11-20)


### Bug Fixes

* resolve YAML linting issues ([06846fc](https://github.com/d0ugal/terraform-provider-garage/commit/06846fc86ac8ec915b7b3acc384f5b5919b6b962))
* resolve YAML linting issues ([7cf1417](https://github.com/d0ugal/terraform-provider-garage/commit/7cf1417fa8d775aa26bc8301e3b9f361136c40bd))

## [0.0.9](https://github.com/d0ugal/terraform-provider-garage/compare/v0.0.8...v0.0.9) (2025-11-17)


### Bug Fixes

* test release ([237a153](https://github.com/d0ugal/terraform-provider-garage/commit/237a15373b880a4ab8a513ec460c16cffab35a88))

## [0.0.8](https://github.com/d0ugal/terraform-provider-garage/compare/v0.0.7...v0.0.8) (2025-11-17)


### Bug Fixes

* test release ([ec3d3bb](https://github.com/d0ugal/terraform-provider-garage/commit/ec3d3bb324814236aaf2c49a08a0345aa4ad832a))

## [0.0.7](https://github.com/d0ugal/terraform-provider-garage/compare/v0.0.6...v0.0.7) (2025-11-17)


### Bug Fixes

* use bash wrapper script for GPG signing with passphrase ([6c2cbc4](https://github.com/d0ugal/terraform-provider-garage/commit/6c2cbc4bed3154c27a2194be7ab0f4c5c526e00f))

## [0.0.6](https://github.com/d0ugal/terraform-provider-garage/compare/v0.0.5...v0.0.6) (2025-11-17)


### Bug Fixes

* provide GPG passphrase to GoReleaser for signing ([54dbc5b](https://github.com/d0ugal/terraform-provider-garage/commit/54dbc5b484f3998830e81a92624b0a6bf4181c62))

## [0.0.5](https://github.com/d0ugal/terraform-provider-garage/compare/v0.0.4...v0.0.5) (2025-11-17)


### Bug Fixes

* configure GPG for non-interactive signing in CI ([4ba876f](https://github.com/d0ugal/terraform-provider-garage/commit/4ba876f8fa423d42a352a31e6aa4b7775e4a9690))

## [0.0.4](https://github.com/d0ugal/terraform-provider-garage/compare/v0.0.3...v0.0.4) (2025-11-17)


### Bug Fixes

* update GoReleaser config to version 2 and fix GPG signing ([f7dc6f7](https://github.com/d0ugal/terraform-provider-garage/commit/f7dc6f70105e036bbd76566447f9fb85e0f5ba20))

## [0.0.3](https://github.com/d0ugal/terraform-provider-garage/compare/v0.0.2...v0.0.3) (2025-11-17)


### Bug Fixes

* lint ([6a6ec1b](https://github.com/d0ugal/terraform-provider-garage/commit/6a6ec1b53b1de72b22505fb605255866532c589c))

## [0.0.2](https://github.com/d0ugal/terraform-provider-garage/compare/v0.0.1...v0.0.2) (2025-11-16)


### Bug Fixes

* test release please ([46ed768](https://github.com/d0ugal/terraform-provider-garage/commit/46ed768bd440ce25f40d4d899fa85101a1e00eda))
