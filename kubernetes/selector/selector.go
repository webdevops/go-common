package selector

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	SelectorError = "<error>"
	SelectorNone  = "<none>"
)

type (
	LabelSelector struct {
		metav1.LabelSelector `json:",inline"`
		selector             *string
	}
)

// IsEmpty checks if the selector is empty/defined or not
func (selector *LabelSelector) IsEmpty() bool {
	if selector == nil || (len(selector.MatchLabels) == 0 && len(selector.MatchExpressions) == 0) {
		return true
	}

	return false
}

// Compile compiles the label selector struct to a string
func (selector *LabelSelector) Compile() (string, error) {
	// no selector
	if selector.IsEmpty() {
		return "", nil
	}

	if selector.selector == nil {
		labelSelector := metav1.FormatLabelSelector(&selector.LabelSelector)

		switch labelSelector {
		case SelectorError:
			return "", fmt.Errorf(`unable to compile Kubernetes selector for resource: %v`, selector)
		case SelectorNone:
			labelSelector = ""
		}

		selector.selector = &labelSelector
	}

	return *selector.selector, nil
}
