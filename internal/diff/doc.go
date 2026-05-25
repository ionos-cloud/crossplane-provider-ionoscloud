// Package diff provides an aggregate, pointer-based diff tracker for
// computing the difference between a desired CRD state (the CR spec) and
// the observed state returned by the IONOS Cloud SDK.
//
// # Design
//
// The Builder is a pure tracker: it records pre-formatted entries but does
// not encode "skip when CR unset" or "skip when SDK nullable returned ok=false"
// rules. Callers (or the complex-structure helper functions in this package)
// guard those cases before invoking a Builder method.
//
// Primitive methods are pointer-typed (Str, Int, Int32, Int64, Float32,
// Float64, Bool). Each compares two pointers and records a diff entry if
// they refer to different values. When one side is nil and the other is not,
// the nil side renders as the sentinel "<nil>".
//
// Complex structures (MaintenanceWindow, Connections, Targets, etc.) get
// dedicated helper functions that take a Builder, a path prefix, the CR-side
// value and the SDK-side pointer. Each helper internally guards the
// "CR unset" case (returning without emitting any diff) and recurses into
// primitive methods for the populated fields.
//
// # Format
//
// Entries are formatted as `path exp=<expected> act=<actual>` and joined by
// `"; "`. Field paths are dot-separated; slice indices appear as `[i]`.
// Examples:
//
//	cores exp=4 act=2; ram exp=8192 act=4096
//	nics[0].ips[1] exp=10.0.0.5 act=10.0.0.6
//	maintenanceWindow.time exp=03:00 act=04:00
//
// Use [Builder.Result] for the canonical IsUpToDate return shape:
//
//	func IsUpToDate(cr *v1alpha1.Server, server sdkgo.Server) (bool, string) {
//	    d := diff.New()
//	    d.Str("name", &cr.Spec.ForProvider.Name, server.Properties.Name)
//	    d.Int32("cores", &cr.Spec.ForProvider.Cores, server.Properties.Cores)
//	    return d.Result()
//	}
//
// # Optional / nullable handling
//
// The "CR field unset → don't diff" rule is the caller's responsibility:
//
//	if cr.Spec.ForProvider.CPUFamily != "" {
//	    d.Str("cpuFamily", &cr.Spec.ForProvider.CPUFamily, server.Properties.CpuFamily)
//	}
//
// For genuinely SDK-nullable fields, the caller invokes the SDK's GetXOk and
// only calls the builder when ok is true:
//
//	if pcc, ok := lan.Properties.GetPccOk(); ok {
//	    d.Str("pcc", &cr.Spec.ForProvider.Pcc, pcc)
//	}
//
// The string-rendering method is [Builder.String], so the primitive method
// for strings is named Str to avoid clashing with fmt.Stringer.
package diff
