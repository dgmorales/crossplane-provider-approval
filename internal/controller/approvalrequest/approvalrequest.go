/*
Copyright 2020 The Crossplane Authors.

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

package approvalrequest

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/crossplane/provider-approval/apis/mock/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-approval/apis/v1alpha1"
	mockclient "github.com/crossplane/provider-approval/internal/client"
)

const (
	errNotApprovalRequest = "managed resource is not a ApprovalRequest custom resource"
	errTrackPCUsage       = "cannot track ProviderConfig usage"
	errGetPC              = "cannot get ProviderConfig"
	errGetCreds           = "cannot get credentials"

	errNewClient = "cannot create new Service"
)

func newMockClient(creds []byte) (*mockclient.Client, error) {
	client := mockclient.Client{
		Hostname: "http://localhost:5000",
	}
	return &client, nil
}

// Setup adds a controller that reconciles ApprovalRequest managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter) error {
	name := managed.ControllerName(v1alpha1.ApprovalRequestGroupKind)

	o := controller.Options{
		RateLimiter: ratelimiter.NewDefaultManagedRateLimiter(rl),
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.ApprovalRequestGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: newMockClient}),
		managed.WithLogger(l.WithValues("controller", name)),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o).
		For(&v1alpha1.ApprovalRequest{}).
		Complete(r)
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube         client.Client
	usage        resource.Tracker
	newServiceFn func(creds []byte) (*mockclient.Client, error)
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.ApprovalRequest)
	if !ok {
		return nil, errors.New(errNotApprovalRequest)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &apisv1alpha1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	cd := pc.Spec.Credentials
	data, err := resource.CommonCredentialExtractor(ctx, cd.Source, c.kube, cd.CommonCredentialSelectors)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	svc, err := c.newServiceFn(data)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{service: svc}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	// A 'client' used to connect to the external resource API. In practice this
	// would be something like an AWS SDK client.
	service *mockclient.Client
	kube    client.Client
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.ApprovalRequest)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotApprovalRequest)
	}

	// These fmt statements should be removed in the real implementation.
	fmt.Printf("Observing: %+v", cr)

	if cr.Status.AtProvider.ID == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	ar, err := c.service.Get(*cr.Status.AtProvider.ID)
	if err != nil {
		return managed.ExternalObservation{ResourceExists: true}, errors.Wrapf(err, "error getting approval request id %d details", *&cr.Status.AtProvider.ID)
	}

	if ar.Archived {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	if cr.Status.AtProvider.Status == v1alpha1.ApprovalStatusValues.Pending && ar.Status == mockclient.ApprovalStatusValues.Approved {
		cr.Status.AtProvider.Signoff = fmt.Sprintf("%s - approved", time.Now().Format("January 2, 2006 at 3:04:05PM MST"))
		cr.SetConditions(xpv1.Available())
	}

	cr.Status.AtProvider.Status = v1alpha1.ApprovalStatus(ar.Status)

	return managed.ExternalObservation{
		// Return false when the external resource does not exist. This lets
		// the managed resource reconciler know that it needs to call Create to
		// (re)create the resource, or that it has successfully been deleted.
		ResourceExists: true,

		// Return false when the external resource exists, but it not up to date
		// with the desired managed resource state. This lets the managed
		// resource reconciler know that it needs to call Update.
		ResourceUpToDate: true,

		// Return any details that may be required to connect to the external
		// resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.ApprovalRequest)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotApprovalRequest)
	}

	fmt.Printf("Creating: %+v", cr)
	// get info from
	ar, err := c.service.Create(cr.Spec.ForProvider.Requester, cr.Spec.ForProvider.Subject)
	if err != nil {
		return managed.ExternalCreation{}, errors.New("failed to create approval request")
	}

	cr.Status.AtProvider.ID = &ar.Id
	cr.Status.AtProvider.Url = fmt.Sprintf("%s/approval_requests/%d", c.service.Hostname, ar.Id)
	cr.Status.AtProvider.Status = v1alpha1.ApprovalStatus(ar.Status)

	cr.SetConditions(xpv1.Creating())

	return managed.ExternalCreation{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.ApprovalRequest)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotApprovalRequest)
	}

	fmt.Printf("Updating (THIS SHOULD NOT BE CALLED): %+v", cr)

	return managed.ExternalUpdate{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.ApprovalRequest)
	if !ok {
		return errors.New(errNotApprovalRequest)
	}

	fmt.Printf("Deleting: %+v", cr)
	_, err := c.service.Archive(*cr.Status.AtProvider.ID)
	if err != nil {
		return errors.Wrapf(err, "error deleting (archiving) approval request id %d", *&cr.Status.AtProvider.ID)
	}

	cr.SetConditions(xpv1.Deleting())

	return nil
}
