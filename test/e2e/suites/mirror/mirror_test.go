// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package mirror_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Successful",
	Label("mirror"),
	func() {
		It("pull non-existent image from mirror should show not found error",
			func(ctx SpecContext) {
				pod, err := kindCluster.Client().CoreV1().Pods(metav1.NamespaceDefault).
					Create(
						ctx,
						&corev1.Pod{
							ObjectMeta: metav1.ObjectMeta{GenerateName: "pod-example-"},
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{{
									Name: "nonexistent",
									Image: fmt.Sprintf(
										"%s/nonexistent:v1.2.3",
										mirrorRegistry.Address(),
									),
									ImagePullPolicy: corev1.PullAlways,
								}},
							},
						},
						metav1.CreateOptions{},
					)
				Expect(err).NotTo(HaveOccurred())

				DeferCleanup(func(ctx SpecContext) {
					err := kindCluster.Client().CoreV1().Pods(metav1.NamespaceDefault).
						Delete(ctx, pod.GetName(), *metav1.NewDeleteOptions(0))
					Expect(err).NotTo(HaveOccurred())
				}, NodeTimeout(time.Second*5))

				Eventually(func(ctx context.Context) string {
					pod, err = kindCluster.Client().CoreV1().Pods(metav1.NamespaceDefault).
						Get(ctx, pod.GetName(), metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					if len(pod.Status.ContainerStatuses) == 0 {
						return ""
					}
					return pod.Status.ContainerStatuses[0].State.Waiting.Reason
				}, time.Minute, time.Second).WithContext(ctx).
					Should(Or(Equal("ErrImagePull"), Equal("ImagePullBackOff")))

				Expect(
					pod.Status.ContainerStatuses[0].State.Waiting.Message,
				).To(HaveSuffix("not found"))
			})
	})
