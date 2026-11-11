package model

import "testing"

// These tests cover one case per kind of attribute value, plus the fallthrough behaviour. They
// deliberately do not enumerate every constant of every enum: that data comes from the platform,
// and transcribing it here would make this file fail whenever the platform legitimately changes an
// enum.

// TaskGroupStatus carries three boolean attributes, so each must map to the right accessor.
func TestTaskGroupStatusAttributes(t *testing.T) {
	tests := []struct {
		status      TaskGroupStatus
		terminating bool
		terminated  bool
		active      bool
	}{
		{TaskGroupStatusPending, false, false, false},
		{TaskGroupStatusRunning, false, false, true},
		{TaskGroupStatusFinishing, false, false, true},
		{TaskGroupStatusFailing, true, false, false},
		{TaskGroupStatusCancelled, false, true, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.Terminating(); got != tt.terminating {
				t.Errorf("Terminating() = %v, want %v", got, tt.terminating)
			}
			if got := tt.status.Terminated(); got != tt.terminated {
				t.Errorf("Terminated() = %v, want %v", got, tt.terminated)
			}
			if got := tt.status.Active(); got != tt.active {
				t.Errorf("Active() = %v, want %v", got, tt.active)
			}
		})
	}
}

// A string-valued attribute, and a constant whose value is absent: Go strings aren't nullable, so
// the absent value becomes the zero value.
func TestStringValuedEnumAttributes(t *testing.T) {
	if got := CloudProviderGoogle.DisplayName(); got != "Google Cloud" {
		t.Errorf("CloudProviderGoogle.DisplayName() = %q, want %q", got, "Google Cloud")
	}
	if got := AttributeSourceTypeSource.Prefix(); got != "source." {
		t.Errorf("AttributeSourceTypeSource.Prefix() = %q, want %q", got, "source.")
	}
	if got := AttributeSourceTypeExternal.Prefix(); got != "" {
		t.Errorf("AttributeSourceTypeExternal.Prefix() = %q, want empty", got)
	}
}

// Permission.Scope is the only attribute whose value is another enum's constant.
func TestEnumValuedEnumAttribute(t *testing.T) {
	if got := PermissionNamespaceRead.Scope(); got != PermissionScopeNamespacedOrGlobal {
		t.Errorf("PermissionNamespaceRead.Scope() = %q, want %q", got, PermissionScopeNamespacedOrGlobal)
	}
	if got := PermissionObjectWrite.Scope(); got != PermissionScopeGlobalOnly {
		t.Errorf("PermissionObjectWrite.Scope() = %q, want %q", got, PermissionScopeGlobalOnly)
	}
}

// A value the SDK doesn't recognise — a status added to the platform after this SDK was
// generated — must fall through to the zero value rather than report something misleading.
func TestUnrecognisedEnumValuesReturnZeroAttributes(t *testing.T) {
	if TaskStatus("NOT_A_REAL_STATUS").Finished() {
		t.Error("unrecognised TaskStatus should not report Finished()")
	}
	if TaskGroupStatus("NOT_A_REAL_STATUS").Active() {
		t.Error("unrecognised TaskGroupStatus should not report Active()")
	}
	if got := CloudProvider("NOT_A_REAL_PROVIDER").DisplayName(); got != "" {
		t.Errorf("unrecognised CloudProvider DisplayName() = %q, want empty", got)
	}
	if got := Permission("NOT_A_REAL_PERMISSION").Scope(); got != "" {
		t.Errorf("unrecognised Permission Scope() = %q, want empty", got)
	}
}
