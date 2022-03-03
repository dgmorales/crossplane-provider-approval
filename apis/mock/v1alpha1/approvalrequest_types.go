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

package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// ApprovalStatus is a string enum for mock approval supported approval status values
type ApprovalStatus string

type approvalStatusValuesType struct {
	Approved ApprovalStatus
	Rejected ApprovalStatus
	Pending  ApprovalStatus
}

// ApprovalStatusValues holds the possible values for mock approval status
var ApprovalStatusValues = approvalStatusValuesType{
	Approved: "Approved",
	Rejected: "Rejected",
	Pending:  "Pending",
}

// ApprovalDecision is a string enum for mock approval supported approval decision values
type ApprovalDecision string

type approvalDecisionValuesType struct {
	Approve ApprovalDecision
	Reject  ApprovalDecision
}

// ApprovalDecisionValues holds the possible values for mock approval decisions
var ApprovalDecisionValues = approvalDecisionValuesType{
	Approve: "approve",
	Reject:  "reject",
}

// ApprovalDecisionRecord records one single approval decision on a approval request
type ApprovalDecisionRecord struct {
	Approver string           `json:"approver"`
	Decision ApprovalDecision `json:"decision"`
}

// ApprovalRequestParameters are the configurable fields of a ApprovalRequest.
type ApprovalRequestParameters struct {
	Requester string `json:"requester"`
	Subject   string `json:"subject"`
}

// ApprovalRequestObservation are the observable fields of a ApprovalRequest.
type ApprovalRequestObservation struct {
	ID        *int                      `json:"id,omitempty"`
	Status    ApprovalStatus           `json:"status,omitempty"`
	Decisions []ApprovalDecisionRecord `json:"decisions,omitempty"`
}

// A ApprovalRequestSpec defines the desired state of a ApprovalRequest.
type ApprovalRequestSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       ApprovalRequestParameters `json:"forProvider"`
}

// A ApprovalRequestStatus represents the observed state of a ApprovalRequest.
type ApprovalRequestStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ApprovalRequestObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A ApprovalRequest is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,approval}
type ApprovalRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApprovalRequestSpec   `json:"spec"`
	Status ApprovalRequestStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ApprovalRequestList contains a list of ApprovalRequest
type ApprovalRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApprovalRequest `json:"items"`
}

// ApprovalRequest type metadata.
var (
	ApprovalRequestKind             = reflect.TypeOf(ApprovalRequest{}).Name()
	ApprovalRequestGroupKind        = schema.GroupKind{Group: Group, Kind: ApprovalRequestKind}.String()
	ApprovalRequestKindAPIVersion   = ApprovalRequestKind + "." + SchemeGroupVersion.String()
	ApprovalRequestGroupVersionKind = SchemeGroupVersion.WithKind(ApprovalRequestKind)
)

func init() {
	SchemeBuilder.Register(&ApprovalRequest{}, &ApprovalRequestList{})
}
