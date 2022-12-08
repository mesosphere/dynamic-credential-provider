[38;5;227m[0m[38;5;227m──────────────────────────────────────────────────────────────────────────────────────────────────────────[0m
[38;5;227mmodified: charts/dynamic-credential-provider/Chart.yaml
[38;5;227m──────────────────────────────────────────────────────────────────────────────────────────────────────────[0m
[1;35m[1;35m@ charts/dynamic-credential-provider/Chart.yaml:3 @[1m[0m
[1;32m[1;32m[m[1;32m# Copyright 2022 D2iQ, Inc. All rights reserved.[m[0m
[1;32m[1;32m[m[1;32m# SPDX-License-Identifier: Apache-2.0[m[0m
[7m[1;32m [m
apiVersion: v2[m
name: dynamic-credential-provider[m
description: A Helm chart for Kubernetes dynamic credential provider[m
[38;5;227m[0m[38;5;227m──────────────────────────────────────────────────────────────────────────────────────────────────────────[0m
[38;5;227mmodified: charts/dynamic-credential-provider/templates/daemonset.yaml
[38;5;227m──────────────────────────────────────────────────────────────────────────────────────────────────────────[0m
[1;35m[1;35m@ charts/dynamic-credential-provider/templates/daemonset.yaml:3 @[1m[0m
[1;32m[1;32m[m[1;32m# Copyright 2022 D2iQ, Inc. All rights reserved.[m[0m
[1;32m[1;32m[m[1;32m# SPDX-License-Identifier: Apache-2.0[m[0m
[7m[1;32m [m
apiVersion: apps/v1[m
kind: DaemonSet[m
metadata:[m
[38;5;227m[0m[38;5;227m──────────────────────────────────────────────────────────────────────────────────────────────────────────[0m
[38;5;227mmodified: charts/dynamic-credential-provider/templates/serviceaccount.yaml
[38;5;227m──────────────────────────────────────────────────────────────────────────────────────────────────────────[0m
[1;35m[1;35m@ charts/dynamic-credential-provider/templates/serviceaccount.yaml:3 @[1m[0m
[1;32m[1;32m[m[1;32m# Copyright 2022 D2iQ, Inc. All rights reserved.[m[0m
[1;32m[1;32m[m[1;32m# SPDX-License-Identifier: Apache-2.0[m[0m
[7m[1;32m [m
{{- if .Values.serviceAccount.create -}}[m
apiVersion: v1[m
kind: ServiceAccount[m
[38;5;227m[0m[38;5;227m──────────────────────────────────────────────────────────────────────────────────────────────────────────[0m
[38;5;227mmodified: charts/dynamic-credential-provider/values.yaml
[38;5;227m──────────────────────────────────────────────────────────────────────────────────────────────────────────[0m
[1;35m[1;35m@ charts/dynamic-credential-provider/values.yaml:3 @[1m[0m
[1;32m[1;32m[m[1;32m# Copyright 2022 D2iQ, Inc. All rights reserved.[m[0m
[1;32m[1;32m[m[1;32m# SPDX-License-Identifier: Apache-2.0[m[0m
[7m[1;32m [m
installer:[m
  kubeletImageCredentialProviderBinDir: /etc/kubernetes/image-credential-provider/[m
[m
