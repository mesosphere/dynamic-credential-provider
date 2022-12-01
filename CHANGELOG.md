# Changelog

<!-- Release notes generated using configuration in .github/release.yaml at main -->

## What's Changed
### Exciting New Features 🎉
* feat: static credentials provider by @dkoshkin in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/3
* feat: Add config type boilerplate by @jimmidyson in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/6
* feat: Add GCR credential provider to installer by @jimmidyson in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/4
* feat: Add mirror config by @jimmidyson in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/8
* feat: Use v1beta1 API for credential provider API by @jimmidyson in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/9
* feat: Reenable upx for macos binaries by @jimmidyson in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/19
* feat: Add functionality for setting the returned credentials by @jimmidyson in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/21
* feat: Copy credential provider config validation from upstream by @jimmidyson in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/22
* feat: Implement shim provider by @jimmidyson in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/23
* feat: Initial demo script using CAPI Docker provider by @jimmidyson in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/29
### Fixes 🔧
* fix: Correctly handle MirrorCredenialsOnly strategy by @jimmidyson in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/25
* fix: Use closest authconfig match by @jimmidyson in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/27
* fix: Handle scheme in mirror endpoint by @jimmidyson in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/28
### Other Changes
* build(deps): Bump github.com/stretchr/testify from 1.8.0 to 1.8.1 by @dependabot in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/2
* refactor: Tidy up static credentials test by @jimmidyson in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/5
* build: Add revive linter by @jimmidyson in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/7
* build(deps): Bump k8s.io/apimachinery from 0.25.3 to 0.25.4 by @dependabot in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/10
* build(deps): Bump k8s.io/kubelet from 0.25.3 to 0.25.4 by @dependabot in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/12
* build: Add default envrc file by @jimmidyson in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/13
* ci: Add dependabot automation to auto-approve and enable auto-merge by @jimmidyson in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/14
* build(deps): Bump sqren/backport-github-action from 8.3.1 to 8.9.7 by @dependabot in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/15
* build(deps): Bump amannn/action-semantic-pull-request from 4 to 5 by @dependabot in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/17
* build(deps): Bump google-github-actions/release-please-action from 3.5 to 3.6 by @dependabot in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/18
* build(deps): Bump github/codeql-action from 1 to 2 by @dependabot in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/16
* ci: Ensure using squash PR for auto-merge dependabot PRs by @jimmidyson in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/20
* refactor: shim demo fixups by @dkoshkin in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/24
* refactor: use 127.0.0.1 in demo script to push images by @dkoshkin in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/26
* test: Add shim provider unit tests by @jimmidyson in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/30
* build(deps): Bump k8s.io/klog/v2 from 2.70.1 to 2.80.1 by @dependabot in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/31
* test: Add initial e2e test by @jimmidyson in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/33
* test(e2e): Add test for falling back to origin by @jimmidyson in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/34
* ci: fix e2e-test in Darwin by always building with GOOS=linux by @dkoshkin in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/35
* test(e2e): Add tests for pulling from mirror by @jimmidyson in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/36
* test(e2e): Fix up e2e binaries on ARM machines by @jimmidyson in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/37
* refactor(e2e): Use ginkgo and gomega functions in helper libs by @jimmidyson in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/38

## New Contributors
* @dependabot made their first contribution in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/2
* @dkoshkin made their first contribution in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/3
* @jimmidyson made their first contribution in https://github.com/mesosphere/kubelet-image-credential-provider-shim/pull/5

**Full Changelog**: https://github.com/mesosphere/kubelet-image-credential-provider-shim/commits/v0.1.0
