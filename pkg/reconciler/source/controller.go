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

	sourceinformer "github.com/GoogleCloudPlatform/kf/pkg/client/injection/informers/kf/v1alpha1/source"
	"github.com/GoogleCloudPlatform/kf/pkg/reconciler"
	"github.com/knative/pkg/configmap"
	"github.com/knative/pkg/controller"
	"github.com/knative/pkg/logging"
)

// NewController creates a new controller capable of reconciling Kf sources.
func NewController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	logger := logging.FromContext(ctx)

	// Get informers off context
	sourceInformer := sourceinformer.Get(ctx)

	// Create reconciler
	c := &Reconciler{
		Base:         reconciler.NewBase(ctx, "source-controller", cmw),
		SourceLister: sourceInformer.Lister(),
	}

	impl := controller.NewImpl(c, logger, "sources")

	c.Logger.Info("Setting up event handlers")

	// Watch for changes in sub-resources so we can sync accordingly
	sourceInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))

	return impl
}
