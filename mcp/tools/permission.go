package tools

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

var (
	// AutoApprove can be enabled for automated tests or environments where stdin is not interactive.
	AutoApprove = false

	// CustomPermissionReader can be overridden during tests.
	CustomPermissionReader io.Reader = nil

	// PermissionHandler allows hooking a custom approval callback.
	// If set and returns (true, nil), action is approved.
	// If returns (false, err), action is rejected.
	PermissionHandler func(toolName, actionDescription string) (bool, error)

	bypassPermissionDepth int
)

// SuppressPermission executes a function while temporarily bypassing permission checks.
// Useful for composite tools (e.g. EditFile or MultiEdit) that internally call ReadFile.
func SuppressPermission(fn func() error) error {
	bypassPermissionDepth++
	defer func() {
		bypassPermissionDepth--
	}()
	return fn()
}

// AskPermission displays the intended action of a tool and prompts the user for approval.
// Returns nil if approved, or an error if denied.
func AskPermission(toolName string, actionDescription string) error {
	// If currently running in a suppressed/nested internal context, bypass prompt
	if bypassPermissionDepth > 0 {
		return nil
	}

	// 1. If custom hook is defined, prioritize it
	if PermissionHandler != nil {
		allowed, err := PermissionHandler(toolName, actionDescription)
		if err != nil {
			return err
		}
		if allowed {
			return nil
		}
		return fmt.Errorf("permission denied by custom handler for tool '%s'", toolName)
	}

	// 2. Display formatted action details to user
	fmt.Println("\n==================================================")
	fmt.Printf("[PERMISSION REQUEST] Tool: %s\n", strings.ToUpper(toolName))
	fmt.Println("Action Details:")
	fmt.Println(actionDescription)
	fmt.Println("--------------------------------------------------")

	// 3. Check AutoApprove
	if AutoApprove {
		fmt.Println("[AUTO-APPROVED] Proceeding without prompt.")
		fmt.Println("==================================================")
		return nil
	}

	// 4. Prompt user from Stdin or CustomPermissionReader
	fmt.Printf("Do you allow %s to proceed? [y/N]: ", toolName)

	var reader *bufio.Reader
	if CustomPermissionReader != nil {
		reader = bufio.NewReader(CustomPermissionReader)
	} else {
		reader = bufio.NewReader(os.Stdin)
	}

	input, err := reader.ReadString('\n')
	if err != nil && len(input) == 0 {
		fmt.Printf("\n[DENIED] Failed to read confirmation: %v\n", err)
		fmt.Println("==================================================")
		return fmt.Errorf("permission denied: unable to read confirmation for tool '%s': %w", toolName, err)
	}

	cleaned := strings.ToLower(strings.TrimSpace(input))
	if cleaned == "y" || cleaned == "yes" {
		fmt.Printf("[APPROVED] Proceeding with %s...\n", toolName)
		fmt.Println("==================================================")
		return nil
	}

	fmt.Printf("[DENIED] Action cancelled by user for %s.\n", toolName)
	fmt.Println("==================================================")
	return fmt.Errorf("permission denied by user for tool '%s'", toolName)
}
