/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	securityv1alpha1 "github.com/DavidHoenisch/coraza-operator/api/v1alpha1"
	"github.com/DavidHoenisch/coraza-operator/internal/sync"
)

var _ = Describe("IPBlockList Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		ipblocklist := &securityv1alpha1.IPBlockList{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind IPBlockList")
			err := k8sClient.Get(ctx, typeNamespacedName, ipblocklist)
			if err != nil && errors.IsNotFound(err) {
				resource := &securityv1alpha1.IPBlockList{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: securityv1alpha1.IPBlockListSpec{
						Sources: []securityv1alpha1.SourceSpec{
							{
								Type: securityv1alpha1.SourceTypeGit,
								Git: &securityv1alpha1.GitSourceSpec{
									URL:  "https://example.com/blocklists.git",
									Path: "lists/blocked-ips.txt",
								},
							},
						},
						OutputSpec: securityv1alpha1.OutputSpec{
							ConfigMapName: "test-blocklist",
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &securityv1alpha1.IPBlockList{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance IPBlockList")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &IPBlockListReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
				Syncer: sync.StaticSyncer{
					Result: sync.Result{
						IPs:       []string{"192.0.2.1", "198.51.100.0/24"},
						CommitSHA: "test-revision",
					},
				},
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			configMap := &corev1.ConfigMap{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-blocklist",
				Namespace: "default",
			}, configMap)
			Expect(err).NotTo(HaveOccurred())
			Expect(configMap.Data).To(HaveKey("blocked-ips.txt"))
			Expect(configMap.Data["blocked-ips.txt"]).NotTo(BeEmpty())

			resource := &securityv1alpha1.IPBlockList{}
			err = k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())
			Expect(resource.Status.BlockIPCount).To(BeNumerically(">", 0))
			Expect(resource.Status.CommitSHA).NotTo(BeEmpty())
		})
	})
})
