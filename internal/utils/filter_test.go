package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func TestDesiredStateChanged(t *testing.T) {
	const ignored, other = "example.com/ignored", "example.com/other"
	object := func(generation int64, labels, annotations map[string]string) *corev1.ConfigMap {
		return &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Generation: generation, Labels: labels, Annotations: annotations}}
	}
	base := map[string]string{ignored: "1", other: "a"}

	for _, tc := range []struct {
		name        string
		before      *corev1.ConfigMap
		after       *corev1.ConfigMap
		want        bool
		wantDefault bool
	}{
		{name: "ignored key added", before: object(1, nil, map[string]string{other: "a"}), after: object(1, nil, base), wantDefault: true},
		{name: "ignored key changed", before: object(1, nil, base), after: object(1, nil, map[string]string{ignored: "2", other: "a"}), wantDefault: true},
		{name: "ignored key removed", before: object(1, nil, base), after: object(1, nil, map[string]string{other: "a"}), want: true, wantDefault: true},
		{name: "other annotation changed", before: object(1, nil, base), after: object(1, nil, map[string]string{ignored: "2", other: "b"}), want: true, wantDefault: true},
		{name: "other annotation removed", before: object(1, nil, base), after: object(1, nil, map[string]string{ignored: "1"}), want: true, wantDefault: true},
		{name: "label changed", before: object(1, nil, base), after: object(1, map[string]string{"l": "v"}, base), want: true, wantDefault: true},
		{name: "generation changed", before: object(1, nil, base), after: object(2, nil, base), want: true, wantDefault: true},
		{name: "status only", before: object(1, nil, base), after: object(1, nil, base)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := event.UpdateEvent{ObjectOld: tc.before, ObjectNew: tc.after}
			assert.Equal(t, tc.want, DesiredStateChanged(ignored).Update(e))
			assert.Equal(t, tc.wantDefault, DesiredStateChanged().Update(e))
		})
	}

	assert.True(t, DesiredStateChanged(ignored).Create(event.CreateEvent{Object: object(1, nil, nil)}))
	assert.True(t, DesiredStateChanged(ignored).Delete(event.DeleteEvent{Object: object(1, nil, nil)}))
}
