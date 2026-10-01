package monitor

import "sync"

// AdminOperationMetrics contains security and reliability counters for the
// privileged control plane. Map keys are bounded route permissions/actions,
// rather than user-controlled values, so the in-memory series cannot grow
// without a corresponding route or permission being added.
type AdminOperationMetrics struct {
	ForbiddenByPermission   map[string]int64 `json:"forbidden_by_permission"`
	FailedMutationsByAction map[string]int64 `json:"failed_mutations_by_action"`
	DestructiveByAction     map[string]int64 `json:"destructive_actions_by_action"`
	ImportFailuresByAction  map[string]int64 `json:"import_failures_by_action"`
	AuditWriteFailures      int64            `json:"audit_write_failures"`
	LegacyAdminFallbacks    int64            `json:"legacy_is_admin_fallbacks"`
	PermissionResolveErrors int64            `json:"permission_resolution_errors"`
}

type AdminOperationTracker struct {
	mu      sync.RWMutex
	metrics AdminOperationMetrics
}

var GlobalAdminOperations = NewAdminOperationTracker()

func NewAdminOperationTracker() *AdminOperationTracker {
	t := &AdminOperationTracker{}
	t.Reset()
	return t
}

func (t *AdminOperationTracker) RecordForbidden(permission string) {
	t.incrementMap(func(m *AdminOperationMetrics) map[string]int64 { return m.ForbiddenByPermission }, permission)
}

func (t *AdminOperationTracker) RecordFailedMutation(action string) {
	t.incrementMap(func(m *AdminOperationMetrics) map[string]int64 { return m.FailedMutationsByAction }, action)
}

func (t *AdminOperationTracker) RecordDestructiveAction(action string) {
	t.incrementMap(func(m *AdminOperationMetrics) map[string]int64 { return m.DestructiveByAction }, action)
}

func (t *AdminOperationTracker) RecordImportFailure(action string) {
	t.incrementMap(func(m *AdminOperationMetrics) map[string]int64 { return m.ImportFailuresByAction }, action)
}

func (t *AdminOperationTracker) incrementMap(selectMap func(*AdminOperationMetrics) map[string]int64, key string) {
	t.mu.Lock()
	selectMap(&t.metrics)[key]++
	t.mu.Unlock()
}

func (t *AdminOperationTracker) RecordAuditWriteFailure() {
	t.mu.Lock()
	t.metrics.AuditWriteFailures++
	t.mu.Unlock()
}

func (t *AdminOperationTracker) RecordLegacyAdminFallback() {
	t.mu.Lock()
	t.metrics.LegacyAdminFallbacks++
	t.mu.Unlock()
}

func (t *AdminOperationTracker) RecordPermissionResolveError() {
	t.mu.Lock()
	t.metrics.PermissionResolveErrors++
	t.mu.Unlock()
}

func (t *AdminOperationTracker) Snapshot() AdminOperationMetrics {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return AdminOperationMetrics{
		ForbiddenByPermission:   cloneCounterMap(t.metrics.ForbiddenByPermission),
		FailedMutationsByAction: cloneCounterMap(t.metrics.FailedMutationsByAction),
		DestructiveByAction:     cloneCounterMap(t.metrics.DestructiveByAction),
		ImportFailuresByAction:  cloneCounterMap(t.metrics.ImportFailuresByAction),
		AuditWriteFailures:      t.metrics.AuditWriteFailures,
		LegacyAdminFallbacks:    t.metrics.LegacyAdminFallbacks,
		PermissionResolveErrors: t.metrics.PermissionResolveErrors,
	}
}

func (t *AdminOperationTracker) Reset() {
	t.mu.Lock()
	t.metrics = AdminOperationMetrics{
		ForbiddenByPermission:   map[string]int64{},
		FailedMutationsByAction: map[string]int64{},
		DestructiveByAction:     map[string]int64{},
		ImportFailuresByAction:  map[string]int64{},
	}
	t.mu.Unlock()
}

func cloneCounterMap(source map[string]int64) map[string]int64 {
	result := make(map[string]int64, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
