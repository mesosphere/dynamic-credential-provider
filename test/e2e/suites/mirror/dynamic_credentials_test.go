// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package mirror_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/distribution/reference"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sethvargo/go-password/password"
	"helm.sh/helm/v3/pkg/cli/output"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	applymetav1 "k8s.io/client-go/applyconfigurations/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/cli-utils/pkg/kstatus/status"

	"github.com/mesosphere/dynamic-credential-provider/test/e2e/docker"
	"github.com/mesosphere/dynamic-credential-provider/test/e2e/env"
	"github.com/mesosphere/dynamic-credential-provider/test/e2e/helm"
)

var _ = Describe("Successful",
	Ordered, Serial,
	Label("daemonset"),
	func() {
		BeforeAll(func(ctx SpecContext) {
			By("Pushing project Docker image to registry")
			img, ok := artifacts.SelectDockerImage(
				"ghcr.io/mesosphere/dynamic-credential-provider",
				"linux",
				runtime.GOARCH,
			)
			Expect(ok).To(BeTrue())

			namedImg, err := reference.ParseNormalizedNamed(img.Name)
			Expect(err).ToNot(HaveOccurred())
			pushedImageName := strings.Replace(
				img.Name,
				reference.Domain(namedImg),
				e2eConfig.Registry.HostPortAddress,
				1,
			)

			err = docker.RetagAndPushImage(
				ctx,
				img.Name,
				pushedImageName,
				env.DockerHubUsername(),
				env.DockerHubPassword(),
				e2eConfig.Registry.Username,
				e2eConfig.Registry.Password,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Creating credential provider config secrets with no mirror specified")
			var buf bytes.Buffer
			Expect(configTemplates.ExecuteTemplate(
				&buf,
				"dynamic-credential-provider-config.yaml.tmpl",
				dynamicCredentialProviderConfigData{},
			)).To(Succeed())
			_, err = kindClusterClient.CoreV1().Secrets(metav1.NamespaceSystem).
				Apply(
					ctx,
					&applycorev1.SecretApplyConfiguration{
						TypeMetaApplyConfiguration: applymetav1.TypeMetaApplyConfiguration{
							APIVersion: ptr.To(corev1.SchemeGroupVersion.String()),
							Kind:       ptr.To("Secret"),
						},
						ObjectMetaApplyConfiguration: &applymetav1.ObjectMetaApplyConfiguration{
							Name: ptr.To("dynamiccredentialproviderconfig"),
						},
						StringData: map[string]string{
							"dynamic-credential-provider-config.yaml": buf.String(),
						},
					},
					metav1.ApplyOptions{
						Force:        true,
						FieldManager: "dynamic-credential-provider-e2e",
					},
				)
			Expect(err).NotTo(HaveOccurred())

			buf.Reset()
			Expect(configTemplates.ExecuteTemplate(
				&buf,
				"static-image-credentials.json.tmpl",
				staticImageCredentialsData{
					DockerHubUsername: env.DockerHubUsername(),
					DockerHubPassword: env.DockerHubPassword(),
				},
			)).To(Succeed())
			_, err = kindClusterClient.CoreV1().Secrets(metav1.NamespaceSystem).
				Apply(
					ctx,
					&applycorev1.SecretApplyConfiguration{
						TypeMetaApplyConfiguration: applymetav1.TypeMetaApplyConfiguration{
							APIVersion: ptr.To(corev1.SchemeGroupVersion.String()),
							Kind:       ptr.To("Secret"),
						},
						ObjectMetaApplyConfiguration: &applymetav1.ObjectMetaApplyConfiguration{
							Name: ptr.To("staticcredentialproviderauth"),
						},
						StringData: map[string]string{
							"static-image-credentials.json": buf.String(),
						},
					},
					metav1.ApplyOptions{
						Force:        true,
						FieldManager: "dynamic-credential-provider-e2e",
					},
				)
			Expect(err).NotTo(HaveOccurred())

			By("Installing dynamic provider daemonset")
			release, err := helm.InstallOrUpgrade(
				ctx,
				"dynamic-credential-provider",
				filepath.Join("..", "..", "..", "..", "charts", "dynamic-credential-provider"),
				map[string]any{
					"image": map[string]any{"tag": namedImg.(reference.NamedTagged).Tag()},
				},
				e2eConfig.Kubeconfig,
				metav1.NamespaceSystem,
				GinkgoWriter.Printf,
				time.Minute,
			)
			var releaseYAML bytes.Buffer
			if encodeErr := output.EncodeYAML(&releaseYAML, release); encodeErr != nil {
				err = errors.Join(err, encodeErr)
			} else {
				AddReportEntry("helm release", ReportEntryVisibilityFailureOrVerbose, releaseYAML.String())
			}
			Expect(err).NotTo(HaveOccurred())
		})

		var ds *appsv1.DaemonSet

		BeforeEach(func() {
			ds = nil
		})

		AfterEach(func() {
			if ds != nil {
				var b bytes.Buffer
				err := output.EncodeYAML(&b, ds)
				Expect(err).NotTo(HaveOccurred())
				AddReportEntry(
					"dynamic credential provider daemonset",
					ReportEntryVisibilityFailureOrVerbose,
					b.String(),
				)
			}
		})

		It("daemonset should be running", func(ctx SpecContext) {
			Eventually(func(ctx context.Context) status.Status {
				var err error
				ds, err = kindClusterClient.AppsV1().DaemonSets(metav1.NamespaceSystem).
					Get(ctx, "dynamic-credential-provider", metav1.GetOptions{})
				if err != nil {
					if k8serrors.IsNotFound(err) {
						return status.NotFoundStatus
					}

					Expect(err).NotTo(HaveOccurred())
				}

				if ds.Status.DesiredNumberScheduled == 0 {
					return status.InProgressStatus
				}

				return objStatus(ds, scheme.Scheme)
			}, time.Minute, time.Second).WithContext(ctx).
				Should(Equal(status.CurrentStatus))
		})

		checkCredentialsInContainer := func(ctx context.Context) {
			staticCredsSecretSecret, err := kindClusterClient.CoreV1().
				Secrets(metav1.NamespaceSystem).
				Get(ctx, "staticcredentialproviderauth", metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			jsonBytes, ok := staticCredsSecretSecret.Data["static-image-credentials.json"]
			Expect(ok).To(BeTrue())
			authJSON := string(jsonBytes)

			Eventually(func(ctx context.Context) string {
				contents, err := docker.ReadFileFromContainer(
					ctx,
					kindClusterName+"-control-plane",
					"/etc/kubernetes/image-credential-provider/static-image-credentials.json",
				)

				Expect(err).NotTo(HaveOccurred())

				return contents
			}, 2*time.Minute, time.Second).WithContext(ctx).
				Should(Equal(authJSON))

			dynamicConfigSecret, err := kindClusterClient.CoreV1().Secrets(metav1.NamespaceSystem).
				Get(ctx, "dynamiccredentialproviderconfig", metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			jsonBytes, ok = dynamicConfigSecret.Data["dynamic-credential-provider-config.yaml"]
			Expect(ok).To(BeTrue())
			dynamicConfigJSON := string(jsonBytes)

			Eventually(func(ctx context.Context) string {
				contents, err := docker.ReadFileFromContainer(
					ctx,
					kindClusterName+"-control-plane",
					"/etc/kubernetes/image-credential-provider/dynamic-credential-provider-config.yaml",
				)

				Expect(err).NotTo(HaveOccurred())

				return contents
			}, 2*time.Minute, time.Second).WithContext(ctx).
				Should(Equal(dynamicConfigJSON))
		}

		It(
			"config should be written to node matching specified credentials",
			func(ctx SpecContext) {
				checkCredentialsInContainer(ctx)
			},
		)

		It("pull image from origin should succeed",
			func(ctx SpecContext) {
				pod := runPod(ctx, kindClusterClient, "nginx:stable")

				Eventually(func(ctx context.Context) status.Status {
					pod, err := kindClusterClient.CoreV1().Pods(metav1.NamespaceDefault).
						Get(ctx, pod.GetName(), metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					return objStatus(pod, scheme.Scheme)
				}, time.Minute, time.Second).WithContext(ctx).
					Should(Equal(status.CurrentStatus))
			})

		It("config should be updated on node when secret updated", func(ctx SpecContext) {
			var buf bytes.Buffer
			Expect(configTemplates.ExecuteTemplate(
				&buf,
				"static-image-credentials.json.tmpl",
				staticImageCredentialsData{
					DockerHubUsername: "invalid",
					DockerHubPassword: "credentials",
				},
			)).To(Succeed())
			_, err := kindClusterClient.CoreV1().Secrets(metav1.NamespaceSystem).
				Apply(
					ctx,
					&applycorev1.SecretApplyConfiguration{
						TypeMetaApplyConfiguration: applymetav1.TypeMetaApplyConfiguration{
							APIVersion: ptr.To(corev1.SchemeGroupVersion.String()),
							Kind:       ptr.To("Secret"),
						},
						ObjectMetaApplyConfiguration: &applymetav1.ObjectMetaApplyConfiguration{
							Name: ptr.To("staticcredentialproviderauth"),
						},
						StringData: map[string]string{
							"static-image-credentials.json": buf.String(),
						},
					},
					metav1.ApplyOptions{
						Force:        true,
						FieldManager: "dynamic-credential-provider-e2e",
					},
				)
			Expect(err).NotTo(HaveOccurred())
			checkCredentialsInContainer(ctx)
		}, SpecTimeout(2*time.Minute))

		It("pull image from origin should fail due to invalid credentials",
			func(ctx SpecContext) {
				pod := runPod(ctx, kindClusterClient, "nginx:stable")

				Eventually(func(ctx context.Context) string {
					var err error
					pod, err = kindClusterClient.CoreV1().Pods(metav1.NamespaceDefault).
						Get(ctx, pod.GetName(), metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					if len(pod.Status.ContainerStatuses) == 0 {
						return ""
					}
					return ptr.Deref(
						pod.Status.ContainerStatuses[0].State.Waiting,
						corev1.ContainerStateWaiting{},
					).Reason
				}, time.Minute, time.Second).WithContext(ctx).
					Should(Or(Equal("ErrImagePull"), Equal("ImagePullBackOff")))
			})

		It(
			"config should be updated on node when secret updated with mirror config",
			func(ctx SpecContext) {
				var buf bytes.Buffer
				Expect(configTemplates.ExecuteTemplate(
					&buf,
					"static-image-credentials.json.tmpl",
					staticImageCredentialsData{
						DockerHubUsername: env.DockerHubUsername(),
						DockerHubPassword: env.DockerHubPassword(),
						MirrorAddress:     e2eConfig.Registry.Address,
						MirrorUsername:    e2eConfig.Registry.Username,
						MirrorPassword:    e2eConfig.Registry.Password,
					},
				)).To(Succeed())
				_, err := kindClusterClient.CoreV1().Secrets(metav1.NamespaceSystem).
					Apply(
						ctx,
						&applycorev1.SecretApplyConfiguration{
							TypeMetaApplyConfiguration: applymetav1.TypeMetaApplyConfiguration{
								APIVersion: ptr.To(corev1.SchemeGroupVersion.String()),
								Kind:       ptr.To("Secret"),
							},
							ObjectMetaApplyConfiguration: &applymetav1.ObjectMetaApplyConfiguration{
								Name: ptr.To("staticcredentialproviderauth"),
							},
							StringData: map[string]string{
								"static-image-credentials.json": buf.String(),
							},
						},
						metav1.ApplyOptions{
							Force:        true,
							FieldManager: "dynamic-credential-provider-e2e",
						},
					)
				Expect(err).NotTo(HaveOccurred())

				buf.Reset()

				Expect(configTemplates.ExecuteTemplate(
					&buf,
					"dynamic-credential-provider-config.yaml.tmpl",
					dynamicCredentialProviderConfigData{
						MirrorAddress: e2eConfig.Registry.Address,
					},
				)).To(Succeed())
				_, err = kindClusterClient.CoreV1().Secrets(metav1.NamespaceSystem).
					Apply(
						ctx,
						&applycorev1.SecretApplyConfiguration{
							TypeMetaApplyConfiguration: applymetav1.TypeMetaApplyConfiguration{
								APIVersion: ptr.To(corev1.SchemeGroupVersion.String()),
								Kind:       ptr.To("Secret"),
							},
							ObjectMetaApplyConfiguration: &applymetav1.ObjectMetaApplyConfiguration{
								Name: ptr.To("dynamiccredentialproviderconfig"),
							},
							StringData: map[string]string{
								"dynamic-credential-provider-config.yaml": buf.String(),
							},
						},
						metav1.ApplyOptions{
							Force:        true,
							FieldManager: "dynamic-credential-provider-e2e",
						},
					)
				Expect(err).NotTo(HaveOccurred())

				checkCredentialsInContainer(ctx)
			},
			SpecTimeout(2*time.Minute),
		)

		It("pull image from origin when it does not exist in mirror",
			func(ctx SpecContext) {
				pod := runPod(ctx, kindClusterClient, "nginx:stable")

				Eventually(func(ctx context.Context) status.Status {
					pod, err := kindClusterClient.CoreV1().Pods(metav1.NamespaceDefault).
						Get(ctx, pod.GetName(), metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					return objStatus(pod, scheme.Scheme)
				}, time.Minute, time.Second).WithContext(ctx).
					Should(Equal(status.CurrentStatus))
			})

		It("pull image that only exists in mirror using origin style address",
			func(ctx SpecContext) {
				randomTag, err := password.Generate(6, 0, 0, true, true)
				Expect(err).NotTo(HaveOccurred())

				podMirrorTag := fmt.Sprintf("nginx:%s", randomTag)

				docker.RetagAndPushImage(
					ctx,
					"nginx:stable",
					fmt.Sprintf("%s/library/%s", e2eConfig.Registry.HostPortAddress, podMirrorTag),
					env.DockerHubUsername(),
					env.DockerHubPassword(),
					e2eConfig.Registry.Username,
					e2eConfig.Registry.Password,
				)

				pod := runPod(ctx, kindClusterClient, podMirrorTag)

				Eventually(func(ctx context.Context) status.Status {
					pod, err := kindClusterClient.CoreV1().Pods(metav1.NamespaceDefault).
						Get(ctx, pod.GetName(), metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					return objStatus(pod, scheme.Scheme)
				}, time.Minute, time.Second).WithContext(ctx).
					Should(Equal(status.CurrentStatus))
			})

		It("pull image that only exists in mirror using mirror address",
			func(ctx SpecContext) {
				randomTag, err := password.Generate(6, 0, 0, true, true)
				Expect(err).NotTo(HaveOccurred())

				podMirrorTag := fmt.Sprintf("nginx:%s", randomTag)
				podMirrorName := fmt.Sprintf("%s/%s", e2eConfig.Registry.Address, podMirrorTag)

				docker.RetagAndPushImage(
					ctx,
					"nginx:stable",
					fmt.Sprintf("%s/%s", e2eConfig.Registry.HostPortAddress, podMirrorTag),
					env.DockerHubUsername(),
					env.DockerHubPassword(),
					e2eConfig.Registry.Username,
					e2eConfig.Registry.Password,
				)

				pod := runPod(ctx, kindClusterClient, podMirrorName)

				Eventually(func(ctx context.Context) status.Status {
					pod, err := kindClusterClient.CoreV1().Pods(metav1.NamespaceDefault).
						Get(ctx, pod.GetName(), metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					return objStatus(pod, scheme.Scheme)
				}, time.Minute, time.Second).WithContext(ctx).
					Should(Equal(status.CurrentStatus))
			})
	})
