package utils

import "sigs.k8s.io/controller-runtime/pkg/predicate"

// DesiredStateChanged - we need event filtering, but we can't use the default predicate because we want to be noticed if createfailed appears
func DesiredStateChanged() predicate.Predicate {
	return predicate.Or(
		predicate.AnnotationChangedPredicate{},
		predicate.LabelChangedPredicate{},
		predicate.GenerationChangedPredicate{},
	)
}
