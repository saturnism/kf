// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package source

import (
	"context"
	"reflect"

	"github.com/google/kf/pkg/apis/kf/v1alpha1"
	kflisters "github.com/google/kf/pkg/client/listers/kf/v1alpha1"
	"github.com/google/kf/pkg/reconciler"
	"go.uber.org/zap"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"knative.dev/pkg/logging"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/controller"
)

// Reconciler reconciles an source object with the K8s cluster.
type Reconciler struct {
	*reconciler.Base

	// listers index properties about resources
	SourceLister kflisters.SourceLister
}

// Check that our Reconciler implements controller.Reconciler
var _ controller.Reconciler = (*Reconciler)(nil)

// Reconcile is called by Kubernetes.
func (r *Reconciler) Reconcile(ctx context.Context, key string) error {
	logger := logging.FromContext(ctx)

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	original, err := r.SourceLister.Sources(namespace).Get(name)
	switch {
	case apierrs.IsNotFound(err):
		logger.Errorf("source %q no longer exists\n", name)
		return nil

	case err != nil:
		return err

	case original.GetDeletionTimestamp() != nil:
		return nil
	}

	// Don't modify the informers copy
	toReconcile := original.DeepCopy()

	// Reconcile this copy of the service and then write back any status
	// updates regardless of whether the reconciliation errored out.
	reconcileErr := r.ApplyChanges(ctx, toReconcile)
	if equality.Semantic.DeepEqual(original.Status, toReconcile.Status) {
		// If we didn't change anything then don't call updateStatus.
		// This is important because the copy we loaded from the informer's
		// cache may be stale and we don't want to overwrite a prior update
		// to status with this stale state.

	} else if _, uErr := r.updateStatus(namespace, toReconcile); uErr != nil {
		logger.Warnw("Failed to update Source status", zap.Error(uErr))
		return uErr
	}

	return reconcileErr
}

// ApplyChanges updates the linked resources in the cluster with the current
// status of the source.
func (r *Reconciler) ApplyChanges(ctx context.Context, source *v1alpha1.Source) error {
	// logger := logging.FromContext(ctx)
	return nil
}

func (r *Reconciler) updateStatus(namespace string, desired *v1alpha1.Source) (*v1alpha1.Source, error) {
	actual, err := r.SourceLister.Sources(namespace).Get(desired.Name)
	if err != nil {
		return nil, err
	}

	// If there's nothing to update, just return.
	if reflect.DeepEqual(actual.Status, desired.Status) {
		return actual, nil
	}

	// Don't modify the informers copy.
	existing := actual.DeepCopy()
	existing.Status = desired.Status

	return r.KfClientSet.KfV1alpha1().Sources(namespace).UpdateStatus(existing)
}
