package pid

import (
    "encoding/json"
    "net"
    "net/http"
    "net/http/httptest"
    "os"
    "path/filepath"
    "strconv"
    "testing"
)

// tmpDir returns a clean temporary directory for a test.
func tmpDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "pidtest-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// TestGenerateToken verifies that generateToken produces a 32-character hex string.
func TestGenerateToken(t *testing.T) {
	token := generateToken()
	if len(token) != 32 {
		t.Errorf("expected token length 32, got %d (token: %q)", len(token), token)
	}
	// Verify all characters are valid hex.
	for _, c := range token {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("token contains non-hex character: %c", c)
		}
	}
}

// TestGenerateTokenUniqueness checks that two consecutive tokens differ.
func TestGenerateTokenUniqueness(t *testing.T) {
	a := generateToken()
	b := generateToken()
	if a == b {
		t.Error("two consecutive tokens should not be equal")
	}
}

// TestPidFilePath returns the expected path.
func TestPidFilePath(t *testing.T) {
	dir := tmpDir(t)
	got := pidFilePath(dir)
	want := filepath.Join(dir, pidFileName)
	if got != want {
		t.Errorf("pidFilePath(%q) = %q, want %q", dir, got, want)
	}
}

//  verifies that an unrelated service on the recorded port is not mistaken for the gateway.
func TestWritePidFileHealthPIDMismatch(t *testing.T) {
    mux := http.NewServeMux()
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        // Return PID 99999 — does not match the PID file entry
        json.NewEncoder(w).Encode(map[string]any{"pid": 99999, "status": "ok"})
    })
    srv := httptest.NewServer(mux)
    defer srv.Close()

    host, portStr, _ := net.SplitHostPort(srv.Listener.Addr().String())
    port, _ := strconv.Atoi(portStr)

    dir := tmpDir(t)
    foreign := PidFileData{
        PID:   os.Getpid(), // real running PID so it reaches isGatewayAlive
        Token: "deadbeef12345678deadbeef12345678",
        Port:  port,
        Host:  host,
    }
    raw, _ := json.MarshalIndent(foreign, "", "  ")
    os.WriteFile(filepath.Join(dir, pidFileName), raw, 0o600)

    // Should succeed — health PID (99999) != PID file PID (os.Getpid()) = not our gateway
    data, err := WritePidFile(dir, "127.0.0.1", 18790)
    if err != nil {
        t.Fatalf("WritePidFile should treat health PID mismatch as stale, got error: %v", err)
    }
    if data.PID != os.Getpid() {
        t.Errorf("PID = %d, want %d", data.PID, os.Getpid())
    }
}

//  verifies that isGatewayAlive uses the host from the PID file instead of hardcoding localhost.
func TestWritePidFileNonLocalhostHost(t *testing.T) {
    if !isProcessRunning(os.Getppid()) {
        t.Skip("skipping: parent process not running in this environment")
    }

    mux := http.NewServeMux()
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        // Return parent PID — matches the PID file entry
        json.NewEncoder(w).Encode(map[string]any{"pid": os.Getppid(), "status": "ok"})
    })
    srv := httptest.NewServer(mux)
    defer srv.Close()

    host, portStr, _ := net.SplitHostPort(srv.Listener.Addr().String())
    port, _ := strconv.Atoi(portStr)

    dir := tmpDir(t)
    foreign := PidFileData{
        PID:   os.Getppid(), // parent PID — real, running, but not us
        Token: "deadbeef12345678deadbeef12345678",
        Port:  port,
        Host:  host,
    }
    raw, _ := json.MarshalIndent(foreign, "", "  ")
    os.WriteFile(filepath.Join(dir, pidFileName), raw, 0o600)

    // Should block — PID exists, health responds with matching PID on non-localhost host
    _, err := WritePidFile(dir, "127.0.0.1", 18790)
    if err == nil {
        t.Fatal("WritePidFile should block startup when gateway is genuinely alive on non-localhost host")
    }
}


//verifies that a foreign process reusing a crashed gateway's PID is treated as stale.
func TestWritePidFileForeignPIDReuse(t *testing.T) {
    dir := tmpDir(t)

    // PID 1 (init/systemd) is always running but won't respond on port 19999
    foreign := PidFileData{
        PID:   1,
        Token: "deadbeef12345678deadbeef12345678",
        Port:  19999, // nothing listening here
        Host:  "127.0.0.1",
    }
    raw, _ := json.MarshalIndent(foreign, "", "  ")
    os.WriteFile(filepath.Join(dir, pidFileName), raw, 0o600)

    // Should succeed — foreign PID reuse should be treated as stale
    data, err := WritePidFile(dir, "127.0.0.1", 18790)
    if err != nil {
        t.Fatalf("WritePidFile should treat foreign PID as stale, got error: %v", err)
    }
    if data.PID != os.Getpid() {
        t.Errorf("PID = %d, want %d", data.PID, os.Getpid())
    }
}

// TestWritePidFile creates a PID file and verifies its contents.
func TestWritePidFile(t *testing.T) {
	dir := tmpDir(t)
	data, err := WritePidFile(dir, "127.0.0.1", 18790)
	if err != nil {
		t.Fatalf("WritePidFile failed: %v", err)
	}

	if data.PID != os.Getpid() {
		t.Errorf("PID = %d, want %d", data.PID, os.Getpid())
	}
	if data.Host != "127.0.0.1" {
		t.Errorf("Host = %q, want %q", data.Host, "127.0.0.1")
	}
	if data.Port != 18790 {
		t.Errorf("Port = %d, want %d", data.Port, 18790)
	}
	if len(data.Token) != 32 {
		t.Errorf("Token length = %d, want 32", len(data.Token))
	}

	// Verify the file exists and can be unmarshalled.
	raw, err := os.ReadFile(filepath.Join(dir, pidFileName))
	if err != nil {
		t.Fatalf("failed to read pid file: %v", err)
	}

	var fileData PidFileData
	if err = json.Unmarshal(raw, &fileData); err != nil {
		t.Fatalf("failed to unmarshal pid file: %v", err)
	}
	if fileData.PID != data.PID || fileData.Token != data.Token {
		t.Error("file data mismatch")
	}

	// Verify file permissions (owner-only read/write).
	info, err := os.Stat(filepath.Join(dir, pidFileName))
	if err != nil {
		t.Fatalf("failed to stat pid file: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0o600 {
		t.Errorf("file permission = %o, want 0600", perm)
	}
}

// TestWritePidFileOverwrite writes twice and verifies the PID file is replaced.
func TestWritePidFileOverwrite(t *testing.T) {
	dir := tmpDir(t)

	data1, err := WritePidFile(dir, "0.0.0.0", 18790)
	if err != nil {
		t.Fatalf("first WritePidFile failed: %v", err)
	}

	// Second write should succeed because the PID matches our process.
	data2, err := WritePidFile(dir, "0.0.0.0", 18800)
	if err != nil {
		t.Fatalf("second WritePidFile failed: %v", err)
	}

	if data2.Token == data1.Token {
		t.Error("token should change on re-write")
	}
	if data2.Port != 18800 {
		t.Errorf("Port = %d, want 18800", data2.Port)
	}
}

// TestWritePidFileStalePID writes a PID file with a non-running PID, then
// verifies WritePidFile cleans it up and writes a new one.
func TestWritePidFileStalePID(t *testing.T) {
	dir := tmpDir(t)

	// Write a PID file with a PID that almost certainly doesn't exist.
	stale := PidFileData{PID: 99999999, Token: "deadbeef12345678deadbeef12345678"}
	raw, _ := json.MarshalIndent(stale, "", "  ")
	os.WriteFile(filepath.Join(dir, pidFileName), raw, 0o600)

	data, err := WritePidFile(dir, "127.0.0.1", 18790)
	if err != nil {
		t.Fatalf("WritePidFile with stale PID failed: %v", err)
	}
	if data.PID != os.Getpid() {
		t.Errorf("PID = %d, want %d", data.PID, os.Getpid())
	}
}

// TestReadPidFileWithCheck verifies reading a valid PID file for the current process.
func TestReadPidFileWithCheck(t *testing.T) {
	dir := tmpDir(t)

	// Some sandboxed environments (e.g. macOS test runner) may restrict
	// signal(0), causing isProcessRunning(getpid()) to return false.
	if !isProcessRunning(os.Getpid()) {
		t.Skip("skipping: isProcessRunning(getpid()) is false in this environment")
	}

	written, err := WritePidFile(dir, "127.0.0.1", 18790)
	if err != nil {
		t.Fatalf("WritePidFile failed: %v", err)
	}

	read := ReadPidFileWithCheck(dir)
	if read == nil {
		t.Fatal("ReadPidFileWithCheck returned nil for current process")
	}
	if read.PID != written.PID || read.Token != written.Token {
		t.Error("read data doesn't match written data")
	}
}

// TestReadPidFileWithCheckNonexistent returns nil for missing file.
func TestReadPidFileWithCheckNonexistent(t *testing.T) {
	dir := tmpDir(t)
	data := ReadPidFileWithCheck(dir)
	if data != nil {
		t.Error("expected nil for nonexistent PID file")
	}
}

// TestReadPidFileWithCheckStalePID auto-cleans a PID file whose process is dead.
func TestReadPidFileWithCheckStalePID(t *testing.T) {
	dir := tmpDir(t)

	stale := PidFileData{PID: 99999999, Token: "deadbeef12345678deadbeef12345678"}
	raw, _ := json.MarshalIndent(stale, "", "  ")
	os.WriteFile(filepath.Join(dir, pidFileName), raw, 0o600)

	data := ReadPidFileWithCheck(dir)
	if data != nil {
		t.Error("expected nil for stale PID")
	}

	// File should be cleaned up.
	if _, err := os.Stat(filepath.Join(dir, pidFileName)); !os.IsNotExist(err) {
		t.Error("stale PID file should be removed")
	}
}

// TestReadPidFileWithCheckInvalidFile auto-cleans malformed PID file.
func TestReadPidFileWithCheckInvalidFile(t *testing.T) {
	dir := tmpDir(t)
	path := filepath.Join(dir, pidFileName)
	os.WriteFile(path, []byte("not json"), 0o600)

	data := ReadPidFileWithCheck(dir)
	if data != nil {
		t.Error("expected nil for malformed pid file")
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("malformed PID file should be removed")
	}
}

// TestRemovePidFile removes the PID file for the current process.
func TestRemovePidFile(t *testing.T) {
	dir := tmpDir(t)

	if _, err := WritePidFile(dir, "127.0.0.1", 18790); err != nil {
		t.Fatalf("WritePidFile failed: %v", err)
	}

	RemovePidFile(dir)

	if _, err := os.Stat(filepath.Join(dir, pidFileName)); !os.IsNotExist(err) {
		t.Error("PID file should be removed")
	}
}

// TestRemovePidFileDifferentPID does not remove a PID file owned by another process.
func TestRemovePidFileDifferentPID(t *testing.T) {
	dir := tmpDir(t)

	other := PidFileData{PID: 99999999, Token: "deadbeef12345678deadbeef12345678"}
	raw, _ := json.MarshalIndent(other, "", "  ")
	os.WriteFile(filepath.Join(dir, pidFileName), raw, 0o600)

	RemovePidFile(dir)

	if _, err := os.Stat(filepath.Join(dir, pidFileName)); os.IsNotExist(err) {
		t.Error("PID file should NOT be removed (different PID)")
	}
}

// TestRemovePidFileNonexistent does not error on missing file.
func TestRemovePidFileNonexistent(t *testing.T) {
	dir := tmpDir(t)
	// Should not panic or error.
	RemovePidFile(dir)
}

func TestRemovePidFileIfPID(t *testing.T) {
	dir := tmpDir(t)

	other := PidFileData{PID: 99999999, Token: "deadbeef12345678deadbeef12345678"}
	raw, _ := json.MarshalIndent(other, "", "  ")
	path := filepath.Join(dir, pidFileName)
	os.WriteFile(path, raw, 0o600)

	removed := RemovePidFileIfPID(dir, 99999999)
	if !removed {
		t.Fatal("expected RemovePidFileIfPID to remove matching pid file")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("PID file should be removed for matching expected PID")
	}
}

func TestRemovePidFileIfPIDMismatch(t *testing.T) {
	dir := tmpDir(t)

	other := PidFileData{PID: 99999999, Token: "deadbeef12345678deadbeef12345678"}
	raw, _ := json.MarshalIndent(other, "", "  ")
	path := filepath.Join(dir, pidFileName)
	os.WriteFile(path, raw, 0o600)

	removed := RemovePidFileIfPID(dir, 88888888)
	if removed {
		t.Fatal("expected RemovePidFileIfPID to keep non-matching pid file")
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("PID file should NOT be removed for mismatching expected PID")
	}
}

// TestWritePidFileContainerPID1 verifies that a leftover PID file with PID 1
// (typical container entrypoint) is treated as stale and overwritten.
func TestWritePidFileContainerPID1(t *testing.T) {
	dir := tmpDir(t)

	stale := PidFileData{PID: 1, Token: "deadbeef12345678deadbeef12345678"}
	raw, _ := json.MarshalIndent(stale, "", "  ")
	os.WriteFile(filepath.Join(dir, pidFileName), raw, 0o600)

	data, err := WritePidFile(dir, "127.0.0.1", 18790)
	if err != nil {
		t.Fatalf("WritePidFile should treat PID 1 as stale, got error: %v", err)
	}
	if data.PID != os.Getpid() {
		t.Errorf("PID = %d, want %d", data.PID, os.Getpid())
	}
}

// TestReadPidFileWithCheckContainerPID1 verifies that a leftover PID file
// with PID 1 is treated as stale and cleaned up.
func TestReadPidFileWithCheckContainerPID1(t *testing.T) {
	if os.Getpid() == 1 {
		t.Skip("test not meaningful when running as PID 1")
	}
	dir := tmpDir(t)

	stale := PidFileData{PID: 1, Token: "deadbeef12345678deadbeef12345678"}
	raw, _ := json.MarshalIndent(stale, "", "  ")
	os.WriteFile(filepath.Join(dir, pidFileName), raw, 0o600)

	data := ReadPidFileWithCheck(dir)
	if data != nil {
		t.Error("expected nil for PID 1 leftover")
	}

	if _, err := os.Stat(filepath.Join(dir, pidFileName)); !os.IsNotExist(err) {
		t.Error("PID 1 leftover file should be removed")
	}
}

// TestReadPidFileUnlockedInvalidJSON returns error for malformed content.
func TestReadPidFileUnlockedInvalidJSON(t *testing.T) {
	dir := tmpDir(t)
	path := filepath.Join(dir, pidFileName)
	os.WriteFile(path, []byte("not json"), 0o600)

	_, err := readPidFileUnlocked(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// TestReadPidFileUnlockedInvalidPID returns error for non-positive PID.
func TestReadPidFileUnlockedInvalidPID(t *testing.T) {
	dir := tmpDir(t)
	path := filepath.Join(dir, pidFileName)
	os.WriteFile(path, []byte(`{"pid": -1, "token": "a"}`), 0o600)

	_, err := readPidFileUnlocked(path)
	if err == nil {
		t.Error("expected error for invalid PID")
	}
}
