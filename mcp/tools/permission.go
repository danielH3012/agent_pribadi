package tools

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

var (
	// AutoApprove menyetujui semua aksi tanpa prompt (untuk test / lingkungan non-interaktif).
	AutoApprove = false

	// CustomPermissionReader menggantikan terminal sebagai sumber jawaban (untuk test).
	CustomPermissionReader io.Reader = nil

	// PermissionHandler adalah hook persetujuan kustom. Kalau di-set, ia yang menentukan
	// dan prompt terminal dilewati. (true, nil) = setuju, (false, nil) = tolak, (_, err) = gagal.
	PermissionHandler func(toolName, actionDescription string) (bool, error)

	// Serialisasi prompt supaya tool call paralel tidak saling menimpa di terminal.
	permMu sync.Mutex
)

// readLine membaca satu baris byte demi byte, tanpa buffering berlebih,
// sehingga satu reader (mis. strings.Reader di test) bisa dipakai untuk beberapa prompt.
func readLine(r io.Reader) (string, error) {
	var sb strings.Builder
	buf := make([]byte, 1)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			if buf[0] == '\n' {
				return sb.String(), nil
			}
			sb.WriteByte(buf[0])
		}
		if err != nil {
			if err == io.EOF && sb.Len() > 0 {
				return sb.String(), nil
			}
			return sb.String(), err
		}
	}
}

// AskPermission menampilkan aksi yang akan dijalankan dan meminta persetujuan.
// Mengembalikan nil kalau disetujui, error kalau ditolak.
//
// PENTING: stdout/stdin dipakai oleh MCP (JSON-RPC), jadi prompt tidak boleh lewat keduanya.
// Prompt ditampilkan dan dibaca lewat /dev/tty; log lain ke stderr.
func AskPermission(toolName string, actionDescription string) error {
	permMu.Lock()
	defer permMu.Unlock()

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

	var (
		in  io.Reader
		out io.Writer = os.Stderr
	)

	switch {
	case AutoApprove:
		fmt.Fprintf(out, "[AUTO-APPROVED] %s\n%s\n", strings.ToUpper(toolName), actionDescription)
		return nil
	case CustomPermissionReader != nil:
		in = CustomPermissionReader
	default:
		tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
		if err != nil {
			// Fail closed: tanpa terminal, jangan pernah menyetujui diam-diam.
			return fmt.Errorf("permission denied: no terminal available to confirm tool '%s' "+
				"(set AutoApprove or PermissionHandler): %w", toolName, err)
		}
		defer tty.Close()
		in, out = tty, tty
	}

	fmt.Fprintln(out, "\n==================================================")
	fmt.Fprintf(out, "[PERMISSION REQUEST] Tool: %s\n", strings.ToUpper(toolName))
	fmt.Fprintln(out, "Action Details:")
	fmt.Fprintln(out, actionDescription)
	fmt.Fprintln(out, "--------------------------------------------------")
	fmt.Fprintf(out, "Do you allow %s to proceed? [y/N]: ", toolName)

	input, err := readLine(in)
	if err != nil && input == "" {
		fmt.Fprintf(out, "\n[DENIED] Failed to read confirmation: %v\n", err)
		return fmt.Errorf("permission denied: unable to read confirmation for tool '%s': %w", toolName, err)
	}

	switch strings.ToLower(strings.TrimSpace(input)) {
	case "y", "yes":
		fmt.Fprintf(out, "[APPROVED] Proceeding with %s...\n", toolName)
		return nil
	}

	fmt.Fprintf(out, "[DENIED] Action cancelled by user for %s.\n", toolName)
	return fmt.Errorf("permission denied by user for tool '%s'", toolName)
}
