<!--
 Copyright 2022 D2iQ, Inc. All rights reserved.
 SPDX-License-Identifier: Apache-2.0
 -->

# dynamic-credential-provider

![Version: 0.0.0-dev](https://img.shields.io/badge/Version-0.0.0--dev-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: v0.0.0-dev](https://img.shields.io/badge/AppVersion-v0.0.0--dev-informational?style=flat-square)

A Helm chart for Kubernetes dynamic credential provider

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| jimmidyson | <jimmidyson@gmail.com> | <https://github.com/jimmidyson> |

## Source Code

* <https://github.com/mesosphere/dynamic-credential-provider>

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| configSync.secrets.dynamicCredentialProviderConfig | string | `"dynamiccredentialproviderconfig"` |  |
| configSync.secrets.staticCredentialProvider | string | `"staticcredentialproviderauth"` |  |
| fullnameOverride | string | `""` |  |
| image.pullPolicy | string | `"IfNotPresent"` |  |
| image.repository | string | `"ghcr.io/mesosphere/dynamic-credential-provider"` |  |
| image.tag | string | `""` |  |
| imagePullSecrets | list | `[]` |  |
| installer.kubeletImageCredentialProviderBinDir | string | `"/etc/kubernetes/image-credential-provider/"` |  |
| nameOverride | string | `""` |  |
| nodeSelector | object | `{}` |  |
| podAnnotations | object | `{}` |  |
| podSecurityContext.runAsUser | int | `0` |  |
| resources | object | `{}` |  |
| securityContext.capabilities.drop[0] | string | `"ALL"` |  |
| securityContext.privileged | bool | `true` |  |
| securityContext.readOnlyRootFilesystem | bool | `true` |  |
| serviceAccount.annotations | object | `{}` |  |
| serviceAccount.create | bool | `true` |  |
| serviceAccount.name | string | `""` |  |
| tolerations[0].effect | string | `"NoSchedule"` |  |
| tolerations[0].key | string | `"node-role.kubernetes.io/control-plane"` |  |
| tolerations[0].operator | string | `"Exists"` |  |
| tolerations[1].effect | string | `"NoSchedule"` |  |
| tolerations[1].key | string | `"node-role.kubernetes.io/master"` |  |
| tolerations[1].operator | string | `"Exists"` |  |
