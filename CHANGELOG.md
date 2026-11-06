# Changelog

## [1.3.1](https://github.com/Arsolitt/terraform-provider-garagehq/compare/v1.3.0...v1.3.1) (2026-11-06)


### Bug Fixes

* **bucket_key:** drop the resource when its bucket is gone ([1e29a3f](https://github.com/Arsolitt/terraform-provider-garagehq/commit/1e29a3f2811130c60b679450e0a95b7b357f265a))
* **bucket_key:** remove permissions on update ([4872180](https://github.com/Arsolitt/terraform-provider-garagehq/commit/4872180d0559733f256530dd9cd6fac088aa154b))
* **bucket_key:** remove permissions on update ([f387708](https://github.com/Arsolitt/terraform-provider-garagehq/commit/f3877083c5ec51c5c0b4f06e9209528f6ef74d07))
* **deps:** bump x/net to 0.62.0 past CVE-2026-78661 ([ce48783](https://github.com/Arsolitt/terraform-provider-garagehq/commit/ce48783bd2c6c6e922137e5482e537a6aa50e810))

## [1.3.0](https://github.com/Arsolitt/terraform-provider-garagehq/compare/v1.2.1...v1.3.0) (2026-11-06)


### Features

* add the garage_node data source ([cb02fd1](https://github.com/Arsolitt/terraform-provider-garagehq/commit/cb02fd114792939c58aa9cd0a07cb64f20d55ca5))
* add the garage_node data source ([9ea123b](https://github.com/Arsolitt/terraform-provider-garagehq/commit/9ea123b0d5a21adc1038aeb2bd1d1ef42e8cc909))
* **bucket:** add max_size and max_objects quotas ([2de5ef1](https://github.com/Arsolitt/terraform-provider-garagehq/commit/2de5ef1137feef4a6eb2a7a434326d0052a1bcd6))
* **bucket:** add max_size and max_objects quotas ([5d6ace7](https://github.com/Arsolitt/terraform-provider-garagehq/commit/5d6ace76211b5a3e2e718d38a77290cc78124712))
* **key:** import a key with a known id and secret ([46f8017](https://github.com/Arsolitt/terraform-provider-garagehq/commit/46f80178759725e2c088a853a11fcfa0457c14f2))
* **key:** import a key with a known id and secret ([ec54a9d](https://github.com/Arsolitt/terraform-provider-garagehq/commit/ec54a9dd84bf940d5204addd5f1b2775eb0c87b0))
* support `terraform import` ([3681c38](https://github.com/Arsolitt/terraform-provider-garagehq/commit/3681c3812923e54da8322647b624bfc6ef382a77))
* support terraform import ([89ad148](https://github.com/Arsolitt/terraform-provider-garagehq/commit/89ad1484c83562ed9479996ee81c40798a1ce325))


### Bug Fixes

* **admin_token:** do not set a zero expiration ([496f8a7](https://github.com/Arsolitt/terraform-provider-garagehq/commit/496f8a70e0a5d1bc4321abaa4caa7d757c42eb39))
* **admin_token:** keep the server expiration and normalize RFC3339 ([0e1c366](https://github.com/Arsolitt/terraform-provider-garagehq/commit/0e1c36687332ec4a6d07e599d30d00d3a1f69322))
* **admin_token:** surface the Garage error on read ([79296e6](https://github.com/Arsolitt/terraform-provider-garagehq/commit/79296e62cadd2445ffd8055863fc7d81f75db23e))
* **bucket:** apply global_alias changes ([78712b6](https://github.com/Arsolitt/terraform-provider-garagehq/commit/78712b633d40012cb85c451b7fa89fbbaa19a46e))
* **bucket:** apply global_alias changes ([288d1a4](https://github.com/Arsolitt/terraform-provider-garagehq/commit/288d1a4aa50cf7a60a91aa8dc80d50dbaafb57c2))
* **bucket:** configure lifecycle rules through the admin API ([8b4ab3f](https://github.com/Arsolitt/terraform-provider-garagehq/commit/8b4ab3f493bb39000c2b3bb0f39422826b5c1972))
* **bucket:** configure lifecycle rules through the admin API ([4ceacbc](https://github.com/Arsolitt/terraform-provider-garagehq/commit/4ceacbce4cc333486c5cd4355c8c156267f6d9da))
* **bucket:** make global alias removal idempotent ([6c90890](https://github.com/Arsolitt/terraform-provider-garagehq/commit/6c9089084e6cf6f2f5ed43d6e0efb3e7748c42b9))
* **cluster_layout:** accept staged changes when no role changes ([2adcf63](https://github.com/Arsolitt/terraform-provider-garagehq/commit/2adcf63a3cee737281e53553715a877f988919ef))
* **cluster_layout:** always send `tags` in a role change ([56a9386](https://github.com/Arsolitt/terraform-provider-garagehq/commit/56a938685b91e679c4edb1f4a1b1dbcd9c55e60a))
* **cluster_layout:** harden staging changes and document revert semantics ([6db5093](https://github.com/Arsolitt/terraform-provider-garagehq/commit/6db50939a1e4d0d057e05770f4396fa548e51e31))
* **cluster_layout:** keep the layout on destroy ([c07daa1](https://github.com/Arsolitt/terraform-provider-garagehq/commit/c07daa196045c9bdc807e2b0669c1c5e9733f932))
* **cluster_layout:** keep the prior state when an update fails ([1ea91f4](https://github.com/Arsolitt/terraform-provider-garagehq/commit/1ea91f42f87222daca3410997a20945826110a9f))
* **cluster_layout:** make role changes, drift, import and destroy work ([#16](https://github.com/Arsolitt/terraform-provider-garagehq/issues/16)) ([a449019](https://github.com/Arsolitt/terraform-provider-garagehq/commit/a449019d824299f097e1c4921cc4dcf88285bcfe))
* **cluster_layout:** remove a node that leaves the configuration ([9c934d7](https://github.com/Arsolitt/terraform-provider-garagehq/commit/9c934d7a52a1df8a3a26c8796a7b633a85e11249))
* **cluster_layout:** show `version` as unknown when roles change ([865efbc](https://github.com/Arsolitt/terraform-provider-garagehq/commit/865efbc5818db57723bd5f20119a349977e34f01))
* **cluster_layout:** show a node that left the layout as drift ([e5a68b6](https://github.com/Arsolitt/terraform-provider-garagehq/commit/e5a68b65ee39f664a683514b7d04e30a016beedd))
* **cluster_layout:** stage only the roles that change ([78db270](https://github.com/Arsolitt/terraform-provider-garagehq/commit/78db270280710ab02076720dd453425e402ff408))
* **cluster_layout:** support the documented import ([64a2179](https://github.com/Arsolitt/terraform-provider-garagehq/commit/64a21796776771d6539310d58eb1828ebbfd2b5d))
* **key:** surface Garage API errors and document destroy behavior ([ec553c0](https://github.com/Arsolitt/terraform-provider-garagehq/commit/ec553c0c496efa9365b6ca3faba4ba2cb1a9bf68))
* **node:** make the self lookup explicit and document empty fields ([07f015b](https://github.com/Arsolitt/terraform-provider-garagehq/commit/07f015bb3249893ac918833e07e83b563554a426))

## [1.2.1](https://github.com/Arsolitt/terraform-provider-garagehq/compare/v1.2.0...v1.2.1) (2026-11-05)


### Bug Fixes

* **deps:** bump x/net past CVE-2026-78669 ([f97b121](https://github.com/Arsolitt/terraform-provider-garagehq/commit/f97b121cf73bd5addef59e632705ef8e8459a42a))

## [1.2.0](https://github.com/Arsolitt/terraform-provider-garagehq/compare/v1.1.2...v1.2.0) (2026-09-06)


### Features

* **bucket:** add website_access configuration block ([73856cd](https://github.com/Arsolitt/terraform-provider-garagehq/commit/73856cdf7fa1df0b3752ab42a30cdb9986f9e040))
* **bucket:** add website_access configuration block ([d35d560](https://github.com/Arsolitt/terraform-provider-garagehq/commit/d35d5606357e4625c394adb08a835e1c364a9aa2))


### Bug Fixes

* **bucket:** validate website_access constraints and drop docs when disabled ([71b1782](https://github.com/Arsolitt/terraform-provider-garagehq/commit/71b17829baa8cb2aaf1aa312302da7b8033d5584))

## [1.1.2](https://github.com/Arsolitt/terraform-provider-garagehq/compare/v1.1.1...v1.1.2) (2026-08-30)


### Bug Fixes

* **deps:** update garage-admin-sdk-golang for Garage v2.3.0 routingRules ([d8a0378](https://github.com/Arsolitt/terraform-provider-garagehq/commit/d8a0378c207abd5b808dc13879cbc5ec4e7c0a64))
* **deps:** update garage-admin-sdk-golang for Garage v2.3.0 routingRules ([7005f15](https://github.com/Arsolitt/terraform-provider-garagehq/commit/7005f15ed3954bc7a5f65094c0b0734ca5bb28b3))

## [1.1.1](https://github.com/Arsolitt/terraform-provider-garagehq/compare/v1.1.0...v1.1.1) (2026-08-27)


### Bug Fixes

* **bucket:** avoid nil-deref panic in lifecycle helpers ([#4](https://github.com/Arsolitt/terraform-provider-garagehq/issues/4)) ([9765f25](https://github.com/Arsolitt/terraform-provider-garagehq/commit/9765f2514399b1cbd07139cbd6bf80a5e4c5e91a))
* **deps:** bump grpc, x/net, x/text, x/mod past known CVEs ([5fad5a6](https://github.com/Arsolitt/terraform-provider-garagehq/commit/5fad5a6d71e7bb11cdabd556a46466187dd57119))

## [1.1.0](https://github.com/Arsolitt/terraform-provider-garagehq/compare/v1.0.0...v1.1.0) (2026-02-20)


### Features

* **cluster_layout:** accept capacity as string with unit suffixes ([9c1c870](https://github.com/Arsolitt/terraform-provider-garagehq/commit/9c1c870e73d9e5b0d4be4a3430e2f94064c3ff91))


### Bug Fixes

* **cluster_layout:** resolve oneOf validation issue with Garage v2.2 ([c2c542f](https://github.com/Arsolitt/terraform-provider-garagehq/commit/c2c542fe8e89dd9a6ccad702c1cd2a45f24efd25))

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
