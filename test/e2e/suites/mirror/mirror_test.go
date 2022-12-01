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
	"github.com/sethvargo/go-password/password"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/mesosphere/dynamic-credential-provider/test/e2e/docker"
	"github.com/mesosphere/dynamic-credential-provider/test/e2e/env"
)

var _ = Describe("Successful",
	Label("mirror"),
	func() {
		runPod := func(ctx context.Context, k8sClient kubernetes.Interface, image string) *corev1.Pod {
			pod, err := k8sClient.CoreV1().Pods(metav1.NamespaceDefault).
				Create(
					ctx,
					&corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{GenerateName: "pod-"},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Name:            "container1",
								Image:           image,
								ImagePullPolicy: corev1.PullAlways,
							}},
						},
					},
					metav1.CreateOptions{},
				)
			Expect(err).NotTo(HaveOccurred())

			DeferCleanup(func(ctx SpecContext) {
				err := kindClusterClient.CoreV1().Pods(metav1.NamespaceDefault).
					Delete(ctx, pod.GetName(), *metav1.NewDeleteOptions(0))
				Expect(err).NotTo(HaveOccurred())
			}, NodeTimeout(time.Second*5))

			return pod
		}

		It("pull non-existent image from mirror should show not found error",
			func(ctx SpecContext) {
				pod := runPod(
					ctx,
					kindClusterClient,
					fmt.Sprintf("%s/nonexistent:v1.2.3", e2eConfig.Registry.Address),
				)

				Eventually(func(ctx context.Context) string {
					var err error
					pod, err = kindClusterClient.CoreV1().Pods(metav1.NamespaceDefault).
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

		It("pull image from origin when it does not exist in mirror",
			func(ctx SpecContext) {
				pod := runPod(ctx, kindClusterClient, "nginx:stable")

				Eventually(func(ctx context.Context) corev1.ConditionStatus {
					var err error
					pod, err = kindClusterClient.CoreV1().Pods(metav1.NamespaceDefault).
						Get(ctx, pod.GetName(), metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					for _, c := range pod.Status.Conditions {
						if c.Type == corev1.PodReady {
							return c.Status
						}
					}
					return corev1.ConditionUnknown
				}, time.Minute, time.Second).WithContext(ctx).
					Should(Equal(corev1.ConditionTrue))
			})

		It("pull image that only exists in mirror using origin style address",
			func(ctx SpecContext) {
				randomTag, err := password.Generate(6, 0, 0, true, true)
				Expect(err).NotTo(HaveOccurred())

				podMirrorTag := fmt.Sprintf("nginx:%s", randomTag)

				err = docker.RetagAndPushImage(
					ctx,
					"nginx:stable",
					fmt.Sprintf("%s/library/%s", e2eConfig.Registry.HostPortAddress, podMirrorTag),
					env.DockerHubUsername(),
					env.DockerHubPassword(),
					e2eConfig.Registry.Username,
					e2eConfig.Registry.Password,
				)
				Expect(err).NotTo(HaveOccurred())

				pod := runPod(ctx, kindClusterClient, podMirrorTag)

				Eventually(func(ctx context.Context) corev1.ConditionStatus {
					var err error
					pod, err = kindClusterClient.CoreV1().Pods(metav1.NamespaceDefault).
						Get(ctx, pod.GetName(), metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					for _, c := range pod.Status.Conditions {
						if c.Type == corev1.PodReady {
							return c.Status
						}
					}
					return corev1.ConditionUnknown
				}, time.Minute, time.Second).WithContext(ctx).
					Should(Equal(corev1.ConditionTrue))
			})

		It("pull image that only exists in mirror using mirror address",
			func(ctx SpecContext) {
				randomTag, err := password.Generate(6, 0, 0, true, true)
				Expect(err).NotTo(HaveOccurred())

				podMirrorTag := fmt.Sprintf("nginx:%s", randomTag)
				podMirrorName := fmt.Sprintf("%s/%s", e2eConfig.Registry.Address, podMirrorTag)

				err = docker.RetagAndPushImage(
					ctx,
					"nginx:stable",
					fmt.Sprintf("%s/%s", e2eConfig.Registry.HostPortAddress, podMirrorTag),
					env.DockerHubUsername(),
					env.DockerHubPassword(),
					e2eConfig.Registry.Username,
					e2eConfig.Registry.Password,
				)
				Expect(err).NotTo(HaveOccurred())

				pod := runPod(ctx, kindClusterClient, podMirrorName)

				Eventually(func(ctx context.Context) corev1.ConditionStatus {
					var err error
					pod, err = kindClusterClient.CoreV1().Pods(metav1.NamespaceDefault).
						Get(ctx, pod.GetName(), metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					for _, c := range pod.Status.Conditions {
						if c.Type == corev1.PodReady {
							return c.Status
						}
					}
					return corev1.ConditionUnknown
				}, time.Minute, time.Second).WithContext(ctx).
					Should(Equal(corev1.ConditionTrue))
			})
	})
