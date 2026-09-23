package tools

import (
	"strings"
	"testing"
)

func TestAskPermission_Approved(t *testing.T) {
	// Simulate user typing "y\n"
	CustomPermissionReader = strings.NewReader("y\n")
	defer func() {
		CustomPermissionReader = nil
	}()

	err := AskPermission("bash", "echo hello")
	if err != nil {
		t.Fatalf("expected nil error on approval, got: %v", err)
	}
}

func TestAskPermission_ApprovedYes(t *testing.T) {
	// Simulate user typing "yes\n"
	CustomPermissionReader = strings.NewReader("yes\n")
	defer func() {
		CustomPermissionReader = nil
	}()

	err := AskPermission("read", "read file foo.txt")
	if err != nil {
		t.Fatalf("expected nil error on approval with 'yes', got: %v", err)
	}
}

func TestAskPermission_Denied(t *testing.T) {
	// Simulate user typing "n\n"
	CustomPermissionReader = strings.NewReader("n\n")
	defer func() {
		CustomPermissionReader = nil
	}()

	err := AskPermission("bash", "rm -rf /")
	if err == nil {
		t.Fatalf("expected error on denial, got nil")
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("expected 'permission denied' in error message, got: %v", err)
	}
}

func TestAskPermission_AutoApprove(t *testing.T) {
	AutoApprove = true
	defer func() {
		AutoApprove = false
	}()

	// Even if reader is empty, AutoApprove should allow it
	CustomPermissionReader = strings.NewReader("")
	defer func() {
		CustomPermissionReader = nil
	}()

	err := AskPermission("write", "write file important.go")
	if err != nil {
		t.Fatalf("expected auto-approval without error, got: %v", err)
	}
}

func TestAskPermission_CustomHandler(t *testing.T) {
	handlerCalled := false
	PermissionHandler = func(toolName, actionDescription string) (bool, error) {
		handlerCalled = true
		if toolName == "bash" {
			return false, nil
		}
		return true, nil
	}
	defer func() {
		PermissionHandler = nil
	}()

	// Bash should be denied
	err := AskPermission("bash", "ls")
	if err == nil {
		t.Fatalf("expected bash to be denied by custom handler")
	}

	// Read should be approved
	err = AskPermission("read", "foo.go")
	if err != nil {
		t.Fatalf("expected read to be approved by custom handler, got: %v", err)
	}

	if !handlerCalled {
		t.Errorf("expected custom PermissionHandler to be called")
	}
}

func TestSuppressPermission(t *testing.T) {
	// Reader returns "n\n", but SuppressPermission should bypass it
	CustomPermissionReader = strings.NewReader("n\n")
	defer func() {
		CustomPermissionReader = nil
	}()

	err := SuppressPermission(func() error {
		return AskPermission("read", "internal read")
	})
	if err != nil {
		t.Fatalf("expected SuppressPermission to bypass prompt, got error: %v", err)
	}
}

func TestToolBash_PermissionDenied(t *testing.T) {
	CustomPermissionReader = strings.NewReader("n\n")
	defer func() {
		CustomPermissionReader = nil
	}()

	res, err := Bash("echo 'dangerous'")
	if err == nil {
		t.Fatalf("expected Bash to fail with permission denied, got result: %v", res)
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("expected permission error, got: %v", err)
	}
}
