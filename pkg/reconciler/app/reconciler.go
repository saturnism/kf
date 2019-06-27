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

package app

import (
	"context"

	"github.com/google/kf/pkg/apis/kf/v1alpha1"
	kflisters "github.com/google/kf/pkg/client/listers/kf/v1alpha1"

	"github.com/google/kf/pkg/reconciler"
	"knative.dev/pkg/controller"
)

// Reconciler reconciles an App object with the K8s cluster.
type Reconciler struct {
	*reconciler.Base

	// listers index properties about resources
	appLister kflisters.AppLister
}

// Check that our Reconciler implements controller.Reconciler
var _ controller.Reconciler = (*Reconciler)(nil)

// Reconcile is called by Kubernetes.
func (r *Reconciler) Reconcile(ctx context.Context, key string) error {
	return nil
}

// ApplyChanges updates the linked resources in the cluster with the current
// status of the space.
func (r *Reconciler) ApplyChanges(ctx context.Context, space *v1alpha1.Space) error {
	return nil
}
