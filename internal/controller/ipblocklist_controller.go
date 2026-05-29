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
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	securityv1alpha1 "github.com/DavidHoenisch/coraza-operator/api/v1alpha1"
	"github.com/DavidHoenisch/coraza-operator/internal/sync"
)

const (
	blocklistDataKey      = "blocked-ips.txt"
	defaultPollInterval   = 5 * time.Minute
	statusRequeueInterval = 10 * time.Second
)

// IPBlockListReconciler reconciles a IPBlockList object
type IPBlockListReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Syncer sync.Syncer
}

// +kubebuilder:rbac:groups=security.adversity.dev,resources=ipblocklists,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=security.adversity.dev,resources=ipblocklists/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=security.adversity.dev,resources=ipblocklists/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *IPBlockListReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	blocklist := &securityv1alpha1.IPBlockList{}
	if err := r.Get(ctx, req.NamespacedName, blocklist); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	pollInterval, err := pollIntervalFromSpec(blocklist.Spec.PollInterval)
	if err != nil {
		if statusErr := r.setStatusError(ctx, blocklist, err); statusErr != nil {
			return ctrl.Result{}, statusErr
		}
		return ctrl.Result{RequeueAfter: statusRequeueInterval}, nil
	}

	if blocklist.Spec.OutputSpec.ConfigMapName == "" {
		err := fmt.Errorf("spec.output.configMapName is required")
		if statusErr := r.setStatusError(ctx, blocklist, err); statusErr != nil {
			return ctrl.Result{}, statusErr
		}
		return ctrl.Result{RequeueAfter: statusRequeueInterval}, nil
	}

	syncer := r.Syncer
	if syncer == nil {
		syncer = sync.NewDefaultSyncer(nil)
	}

	syncResult, err := syncer.Fetch(ctx, blocklist.Spec)
	if err != nil {
		if statusErr := r.setStatusError(ctx, blocklist, err); statusErr != nil {
			return ctrl.Result{}, statusErr
		}
		return ctrl.Result{RequeueAfter: pollInterval}, nil
	}

	if err := r.reconcileConfigMap(ctx, blocklist, syncResult); err != nil {
		if statusErr := r.setStatusError(ctx, blocklist, err); statusErr != nil {
			return ctrl.Result{}, statusErr
		}
		return ctrl.Result{RequeueAfter: pollInterval}, nil
	}

	blocklist.Status.LastSync = metav1.Now()
	blocklist.Status.Errors = ""
	blocklist.Status.CommitSHA = syncResult.CommitSHA
	blocklist.Status.BlockIPCount = int64(len(syncResult.IPs))
	if err := r.Status().Update(ctx, blocklist); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("synced blocklist",
		"configMap", blocklist.Spec.OutputSpec.ConfigMapName,
		"ips", len(syncResult.IPs),
		"commit", syncResult.CommitSHA,
	)

	return ctrl.Result{RequeueAfter: pollInterval}, nil
}

func (r *IPBlockListReconciler) reconcileConfigMap(
	ctx context.Context,
	blocklist *securityv1alpha1.IPBlockList,
	syncResult sync.Result,
) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      blocklist.Spec.OutputSpec.ConfigMapName,
			Namespace: blocklist.Namespace,
		},
	}

	content := sync.FormatBlocklist(syncResult.IPs)
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		if cm.Data == nil {
			cm.Data = map[string]string{}
		}
		cm.Data[blocklistDataKey] = content
		return controllerutil.SetControllerReference(blocklist, cm, r.Scheme)
	})
	return err
}

func (r *IPBlockListReconciler) setStatusError(
	ctx context.Context,
	blocklist *securityv1alpha1.IPBlockList,
	syncErr error,
) error {
	blocklist.Status.LastSync = metav1.Now()
	blocklist.Status.Errors = syncErr.Error()
	if err := r.Status().Update(ctx, blocklist); err != nil {
		return err
	}
	return syncErr
}

func pollIntervalFromSpec(raw string) (time.Duration, error) {
	if raw == "" {
		return defaultPollInterval, nil
	}
	return time.ParseDuration(raw)
}

// SetupWithManager sets up the controller with the Manager.
func (r *IPBlockListReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&securityv1alpha1.IPBlockList{}).
		Named("ipblocklist").
		Complete(r)
}
