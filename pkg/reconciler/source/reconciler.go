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
	"sort"

	"github.com/google/kf/pkg/apis/kf/v1alpha1"
	kflisters "github.com/google/kf/pkg/client/listers/kf/v1alpha1"
	"github.com/google/kf/pkg/reconciler"
	"github.com/google/kf/pkg/reconciler/source/resources"
	build "github.com/knative/build/pkg/apis/build/v1alpha1"
	cbuild "github.com/knative/build/pkg/client/clientset/versioned/typed/build/v1alpha1"
	buildlisters "github.com/knative/build/pkg/client/listers/build/v1alpha1"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
)

// Reconciler reconciles an source object with the K8s cluster.
type Reconciler struct {
	*reconciler.Base

	buildClient cbuild.BuildV1alpha1Interface

	// listers index properties about resources
	SourceLister kflisters.SourceLister
	buildLister  buildlisters.BuildLister
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
	case errors.IsNotFound(err):
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
	// Sync build
	{
		desired, err := resources.MakeBuild(source)
		if err != nil {
			return err
		}

		selec := labels.NewSelector()
		rec, err := labels.NewRequirement("kf-source", selection.Equals, []string{source.Name})
		if err != nil {
			return err
		}

		selec = selec.Add(*rec)
		actualList, err := r.buildLister.Builds(desired.Namespace).List(selec)
		if err != nil {
			return err
		}

		var actual *build.Build
		if len(actualList) > 0 {
			// Builds are not always listed in their creation order.
			sort.Slice(actualList, func(i, j int) bool {
				return actualList[i].Name < actualList[j].Name
			})
			actual = actualList[len(actualList)-1]
		}

		if actual == nil {
			actual, err = r.buildClient.Builds(desired.Namespace).Create(desired)
			if err != nil {
				return err
			}
		} else if actual, err = r.reconcileBuild(ctx, desired, actual); err != nil {
			return err
		}

		source.Status.PropagateBuildStatus(actual)
	}

	return nil
}

func (r *Reconciler) reconcileBuild(ctx context.Context, desired, actual *build.Build) (*build.Build, error) {
	// Check for differences, if none we don't need to reconcile.
	semanticEqual := equality.Semantic.DeepEqual(desired.ObjectMeta.Labels, actual.ObjectMeta.Labels)

	existing := actual.DeepCopy()

	desiredName := desired.Name
	existingName := existing.Name

	desiredArgs := desired.Spec.Template.Arguments
	desired.Spec.Template.Arguments = desiredArgs[1:]

	existingArgs := existing.Spec.Template.Arguments
	existing.Spec.Template.Arguments = existingArgs[1:]

	desired.Name = ""
	existing.Name = ""

	semanticEqual = semanticEqual && equality.Semantic.DeepEqual(desired.Spec.Source, existing.Spec.Source)
	semanticEqual = semanticEqual && equality.Semantic.DeepEqual(desired.Spec.Template, existing.Spec.Template)

	desired.Name = desiredName
	existing.Name = existingName

	desired.Spec.Template.Arguments = desiredArgs
	existing.Spec.Template.Arguments = existingArgs

	if semanticEqual {
		return actual, nil
	}

	return r.buildClient.Builds(desired.Namespace).Create(desired)
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
