package utils

import (
	"maps"

	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// DesiredStateChanged - we need event filtering, but we can't use the default predicate because we want to be noticed if createfailed appears.
// Adding or changing one of the ignoredAnnotations does not pass the filter, removing one does.
func DesiredStateChanged(ignoredAnnotations ...string) predicate.Predicate {
	var annotationChanged predicate.Predicate = predicate.AnnotationChangedPredicate{}
	if len(ignoredAnnotations) > 0 {
		annotationChanged = predicate.Funcs{UpdateFunc: func(e event.UpdateEvent) bool {
			return annotationsChangedIgnoring(e, ignoredAnnotations)
		}}
	}
	return predicate.Or(
		annotationChanged,
		predicate.LabelChangedPredicate{},
		predicate.GenerationChangedPredicate{},
	)
}

func annotationsChangedIgnoring(e event.UpdateEvent, ignored []string) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}
	oldAnnotations, newAnnotations := maps.Clone(e.ObjectOld.GetAnnotations()), maps.Clone(e.ObjectNew.GetAnnotations())
	for _, k := range ignored {
		_, inOld := oldAnnotations[k]
		_, inNew := newAnnotations[k]
		if inOld && !inNew {
			return true
		}
		delete(oldAnnotations, k)
		delete(newAnnotations, k)
	}
	return !maps.Equal(oldAnnotations, newAnnotations)
}
