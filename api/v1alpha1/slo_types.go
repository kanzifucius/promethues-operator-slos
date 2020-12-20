/*


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
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

var (
	DefaultSamples = []sample{
		{
			Name:     "short",
			Interval: "30s",
			Buckets:  []string{"5m", "30m", "1h"},
		},
		{
			Name:     "medium",
			Interval: "2m",
			Buckets:  []string{"2h", "6h"},
		},
		{
			Name:     "daily",
			Interval: "5m",
			Buckets:  []string{"1d", "3d"},
		},
	}
)

type sample struct {
	Name     string
	Interval string
	Buckets  []string
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SloSpec defines the desired state of Slo
type SloSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of Slo. Edit Slo_types.go to remove/update

	Objectives Objectives `json:"objectives"`

	// +kubebuilder:validation:Optional
	TrafficRateRecord ExprBlock `json:"trafficRateRecord"`
	// +kubebuilder:validation:Optional
	ErrorRateRecord ExprBlock `json:"errorRateRecord"`
	// +kubebuilder:validation:Optional
	LatencyRecord ExprBlock `json:"latencyRecord"`
	// +kubebuilder:validation:Optional
	LatencyQuantileRecord ExprBlock `json:"latencyQuantileRecord"`
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels"`
	// +kubebuilder:validation:Optional
	Annotations map[string]string `json:"annotations"`
}

// SloStatus defines the observed state of Slo
type SloStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Slo is the Schema for the sloes API
type Slo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SloSpec   `json:"spec,omitempty"`
	Status SloStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SloList contains a list of Slo
type SloList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Slo `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Slo{}, &SloList{})
}

type ExprBlock struct {
	// +kubebuilder:validation:Optional
	AlertMethod string `json:"alertMethod"`
	// +kubebuilder:validation:Optional
	BurnRate string `json:"burnRate"`
	// +kubebuilder:validation:Optional
	Windows []Window `json:"windows"`
	// +kubebuilder:validation:Optional
	ShortWindow *bool `json:"shortWindow"`
	// +kubebuilder:validation:Optional
	Buckets []string `json:"buckets"` // used to define buckets of histogram when using latency expression
	// +kubebuilder:validation:Optional
	Expr string `json:"expr"`
}

type Window struct {
	Duration     string `json:"duration"`
	Consumption  string `json:"consumption"`
	Notification string `json:"notification"`
}

func (block *ExprBlock) GetShortWindow() bool {
	defaultShortWindow := true

	if block.ShortWindow == nil {
		return defaultShortWindow
	}

	return *block.ShortWindow
}
func (block *ExprBlock) ComputeExpr(window, le string) string {
	replacer := strings.NewReplacer("$window", window, "$le", le)
	return replacer.Replace(block.Expr)
}

func (block *ExprBlock) ComputeQuantile(window string, quantile float64) string {
	replacer := strings.NewReplacer("$window", window, "$quantile", fmt.Sprintf("%g", quantile))
	return replacer.Replace(block.Expr)
}

type Objectives struct {
	Availability string          `json:"availability"`
	Latency      []LatencyTarget `json:"latency"`
	Window       string          `json:"window"`
}

type LatencyTarget struct {
	LE     string `json:"le"`
	Target string `json:"target"`
}
