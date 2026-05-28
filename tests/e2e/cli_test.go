package e2e_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hearth runs the hearth binary with the given arguments and returns
// stdout, stderr, and the exit code.
func hearth(t *testing.T, dbPath string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	bin := filepath.Join("..", "..", "bin", "hearth")

	// Build the binary if it doesn't exist
	if _, err := os.Stat(bin); os.IsNotExist(err) {
		build := exec.CommandContext(context.Background(), "go", "build", "-o", bin, "../../cmd/hearth")
		build.Dir = "."
		if out, err := build.CombinedOutput(); err != nil {
			t.Fatalf("failed to build hearth: %s\n%s", err, out)
		}
	}

	// Use a per-test config file so tests don't share household_id state.
	configPath := dbPath + ".config.yaml"

	cmd := exec.CommandContext(context.Background(), bin, args...)
	cmd.Env = append(os.Environ(),
		"HEARTH_DB="+dbPath,
		"HEARTH_CONFIG="+configPath,
	)

	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}
	return
}

func TestCLI_Version_PrintsVersion(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "test.db")
	stdout, _, code := hearth(t, db, "version")
	assert.Equal(t, 0, code)
	assert.Contains(t, stdout, "hearth")
}

func TestCLI_Help_IsCleanAndDocumentsFlags(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "test.db")
	stdout, _, code := hearth(t, db, "--help")
	assert.Equal(t, 0, code)
	assert.Contains(t, stdout, "--output")
	assert.Contains(t, stdout, "--household")
	assert.Contains(t, stdout, "accounts")
	assert.Contains(t, stdout, "transactions")
	assert.Contains(t, stdout, "report")
}

func TestCLI_Init_CreatesDatabase(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "test.db")

	stdout, _, code := hearth(t, db, "init", "--name", "Test Household")
	assert.Equal(t, 0, code, "stderr: %s", stdout)
	assert.Contains(t, stdout, "Initialized Hearth database")
	assert.Contains(t, stdout, "Test Household")

	_, err := os.Stat(db)
	require.NoError(t, err, "database file should exist after init")
}

func TestCLI_Init_IdempotentSecondRun(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "test.db")

	_, _, code := hearth(t, db, "init", "--name", "First")
	require.Equal(t, 0, code)

	stdout, _, code := hearth(t, db, "init", "--name", "Second")
	assert.Equal(t, 0, code)
	assert.Contains(t, stdout, "already exists")
}

func TestCLI_AccountsAdd_CreatesAccount(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "test.db")

	// Need to init first to have a household
	_, _, code := hearth(t, db, "init", "--name", "Test HH")
	require.Equal(t, 0, code)

	// Override the household ID used by accounts add
	// (read it from stdout of init)
	stdout, _, code := hearth(t, db, "accounts", "add", "--name", "Checking", "--type", "asset")
	assert.Equal(t, 0, code, "stderr: %s", stdout)
	assert.Contains(t, stdout, "Account created")
	assert.Contains(t, stdout, "Checking")
}

func TestCLI_AccountsList_ReturnsTableOutput(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "test.db")
	setupDB(t, db)

	stdout, _, code := hearth(t, db, "accounts", "list")
	assert.Equal(t, 0, code)
	assert.Contains(t, stdout, "ID")
	assert.Contains(t, stdout, "NAME")
}

func TestCLI_AccountsList_JSONOutput(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "test.db")
	setupDB(t, db)

	stdout, _, code := hearth(t, db, "--output", "json", "accounts", "list")
	assert.Equal(t, 0, code)
	assert.Contains(t, stdout, `"name"`)
	assert.Contains(t, stdout, `"type"`)
}

func TestCLI_TransactionsAdd_UnbalancedEntry_ExitsWithCode3(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "test.db")
	setupDB(t, db)

	_, stderr, code := hearth(t, db,
		"transactions", "add",
		"--description", "Unbalanced",
		"--posting", "acc-checking:100.00:USD",
		"--posting", "acc-expenses:50.00:USD", // doesn't balance
	)
	assert.Equal(t, 3, code, "expected exit code 3 for GAAP violation")
	assert.Contains(t, stderr, "GAAP")
}

func TestCLI_TransactionsAdd_BalancedEntry_Succeeds(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "test.db")
	hhID, checkingID, expenseID := setupDB(t, db)
	_ = hhID

	stdout, _, code := hearth(t, db,
		"--household", hhID,
		"transactions", "add",
		"--description", "Grocery run",
		"--date", "2025-06-15",
		"--posting", checkingID+":-50.00:USD",
		"--posting", expenseID+":50.00:USD",
	)
	assert.Equal(t, 0, code, "stderr: %s", stdout)
	assert.Contains(t, stdout, "Transaction recorded")
}

func TestCLI_TransactionsList_ReturnsEntries(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "test.db")
	hhID, checkingID, expenseID := setupDB(t, db)

	_, _, code := hearth(t, db,
		"--household", hhID,
		"transactions", "add",
		"--description", "Test entry",
		"--posting", checkingID+":-30.00:USD",
		"--posting", expenseID+":30.00:USD",
	)
	require.Equal(t, 0, code)

	stdout, _, code := hearth(t, db, "--household", hhID, "transactions", "list")
	assert.Equal(t, 0, code)
	assert.Contains(t, stdout, "Test entry")
}

func TestCLI_ReportBalance_ShowsAccounts(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "test.db")
	hhID, _, _ := setupDB(t, db)

	stdout, _, code := hearth(t, db, "--household", hhID, "report", "balance")
	assert.Equal(t, 0, code)
	assert.Contains(t, stdout, "Balance Sheet")
}

// setupDB initialises a database with a household and two accounts.
// Returns householdID, checkingAccountID, expenseAccountID.
func setupDB(t *testing.T, db string) (hhID, checkingID, expenseID string) {
	t.Helper()

	// init
	stdout, _, code := hearth(t, db, "init", "--name", "Test HH")
	require.Equal(t, 0, code, stdout)

	// Extract household ID from stdout ("Household: Test HH (ID: <uuid>)")
	for _, line := range strings.Split(stdout, "\n") {
		if strings.HasPrefix(line, "Household:") {
			parts := strings.Split(line, "(ID: ")
			if len(parts) == 2 {
				hhID = strings.TrimSuffix(strings.TrimSpace(parts[1]), ")")
			}
		}
	}
	require.NotEmpty(t, hhID, "could not parse household ID from init output")

	// add accounts
	stdout, _, code = hearth(t, db, "--household", hhID, "accounts", "add", "--name", "Checking", "--type", "asset")
	require.Equal(t, 0, code)
	checkingID = extractIDFromAddOutput(stdout)

	stdout, _, code = hearth(t, db, "--household", hhID, "accounts", "add", "--name", "Expenses", "--type", "expense")
	require.Equal(t, 0, code)
	expenseID = extractIDFromAddOutput(stdout)

	return
}

// extractIDFromAddOutput parses "Account created: NAME (ID: <uuid>)".
func extractIDFromAddOutput(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if strings.Contains(line, "(") && strings.Contains(line, ")") {
			parts := strings.Split(line, "(")
			for _, p := range parts[1:] {
				if strings.HasPrefix(p, "ID: ") || strings.HasPrefix(p, "id: ") {
					return strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(p, "id: "), "ID: "), ")")
				}
			}
		}
	}
	return ""
}
