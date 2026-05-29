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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SourceType identifies which fetcher implementation should handle a source.
// +kubebuilder:validation:Enum=Git;HTTP
type SourceType string

const (
	SourceTypeGit  SourceType = "Git"
	SourceTypeHTTP SourceType = "HTTP"
)

// GitSourceSpec configures a Git repository blocklist source.
type GitSourceSpec struct {
	URL    string `json:"url"`
	Branch string `json:"branch,omitempty"`
	Path   string `json:"path"`
}

// HTTPSourceSpec configures a remote HTTP blocklist source.
type HTTPSourceSpec struct {
	URL string `json:"url"`
}

// SourceSpec declares one blocklist source. Exactly one nested config must
// match the configured type.
type SourceSpec struct {
	Type SourceType      `json:"type"`
	Git  *GitSourceSpec  `json:"git,omitempty"`
	HTTP *HTTPSourceSpec `json:"http,omitempty"`
}

type OutputSpec struct {
	ConfigMapName string `json:"configMapName"`
}

// IPBlockListSpec defines the desired state of IPBlockList.
type IPBlockListSpec struct {
	Sources      []SourceSpec `json:"sources"`
	PollInterval string       `json:"pollInterval,omitempty"`
	OutputSpec   OutputSpec   `json:"output"`
}

// IPBlockListStatus defines the observed state of IPBlockList.
type IPBlockListStatus struct {
	LastSync     metav1.Time `json:"lastSync,omitempty"`
	Errors       string      `json:"error,omitempty"`
	CommitSHA    string      `json:"commitSHA,omitempty"`
	BlockIPCount int64       `json:"blockIpCount,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// IPBlockList is the Schema for the ipblocklists API.
type IPBlockList struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IPBlockListSpec   `json:"spec,omitempty"`
	Status IPBlockListStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IPBlockListList contains a list of IPBlockList.
type IPBlockListList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IPBlockList `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IPBlockList{}, &IPBlockListList{})
}
