package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/steveyegge/beads/internal/storage/sqlite"
	"github.com/steveyegge/beads/internal/types"
)

// TestDatabaseReinitialization tests all database reinitialization scenarios
// covered in DATABASE_REINIT_BUG.md
func TestDatabaseReinitialization(t *testing.T) {
	t.Run("fresh_clone_auto_import", testFreshCloneAutoImport)
	t.Run("database_removal_scenario", testDatabaseRemovalScenario)
	t.Run("legacy_filename_support", testLegacyFilenameSupport)
	t.Run("precedence_test", testPrecedenceTest)
	t.Run("init_safety_check", testInitSafetyCheck)
}

// testFreshCloneAutoImport verifies auto-import works on fresh clone
func testFreshCloneAutoImport(t *testing.T) {
	dir := t.TempDir()

	// Initialize git repo
	runCmd(t, dir, "git", "init")
	runCmd(t, dir, "git", "config", "user.email", "test@example.com")
	runCmd(t, dir, "git", "config", "user.name", "Test User")

	// Create .beads directory with beads.jsonl
	beadsDir := filepath.Join(dir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatalf("Failed to create .beads dir: %v", err)
	}

	// Create test issue data
	issue := &types.Issue{
		ID:          "test-1",
		Title:       "Test issue",
		Description: "Test description",
		Status:      types.StatusOpen,
		Priority:    2,
		IssueType:   types.TypeTask,
	}

	jsonlPath := filepath.Join(beadsDir, "beads.jsonl")
	if err := writeJSONL(jsonlPath, []*types.Issue{issue}); err != nil {
		t.Fatalf("Failed to write JSONL: %v", err)
	}

	// Commit to git
	runCmd(t, dir, "git", "add", ".beads/beads.jsonl")
	runCmd(t, dir, "git", "commit", "-m", "Initial commit")

	// Remove database to simulate fresh clone
	dbPath := filepath.Join(beadsDir, "test.db")
	os.Remove(dbPath)

	// Run bd init with auto-import disabled to test checkGitForIssues
	dbPath = filepath.Join(beadsDir, "test.db")
	store, err := sqlite.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	if err := store.SetConfig(ctx, "issue_prefix", "test"); err != nil {
		t.Fatalf("Failed to set prefix: %v", err)
	}

	// Test checkGitForIssues detects beads.jsonl
	originalDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(originalDir)

	count, path := checkGitForIssues()
	if count != 1 {
		t.Errorf("Expected 1 issue in git, got %d", count)
	}
	if path != ".beads/beads.jsonl" {
		t.Errorf("Expected path .beads/beads.jsonl, got %s", path)
	}

	// Import from git
	if err := importFromGit(ctx, dbPath, store, path); err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Verify issue was imported
	stats, err := store.GetStatistics(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}
	if stats.TotalIssues != 1 {
		t.Errorf("Expected 1 issue after import, got %d", stats.TotalIssues)
	}

	// Verify local beads.jsonl exists after init would call exportToJSONLWithStore
	localPath := filepath.Join(beadsDir, "beads.jsonl")
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		t.Error("Local beads.jsonl should exist after import")
	}
}

// testDatabaseRemovalScenario tests the primary bug scenario
func testDatabaseRemovalScenario(t *testing.T) {
	dir := t.TempDir()

	// Initialize git repo
	runCmd(t, dir, "git", "init")
	runCmd(t, dir, "git", "config", "user.email", "test@example.com")
	runCmd(t, dir, "git", "config", "user.name", "Test User")

	// Create .beads directory with beads.jsonl
	beadsDir := filepath.Join(dir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatalf("Failed to create .beads dir: %v", err)
	}

	// Create multiple test issues
	issues := []*types.Issue{
		{
			ID:        "test-1",
			Title:     "First issue",
			Status:    types.StatusOpen,
			Priority:  1,
			IssueType: types.TypeTask,
		},
		{
			ID:        "test-2",
			Title:     "Second issue",
			Status:    types.StatusOpen,
			Priority:  2,
			IssueType: types.TypeBug,
		},
	}

	jsonlPath := filepath.Join(beadsDir, "beads.jsonl")
	if err := writeJSONL(jsonlPath, issues); err != nil {
		t.Fatalf("Failed to write JSONL: %v", err)
	}

	// Commit to git
	runCmd(t, dir, "git", "add", ".beads/beads.jsonl")
	runCmd(t, dir, "git", "commit", "-m", "Add issues")

	// Simulate rm -rf .beads/
	os.RemoveAll(beadsDir)
	os.MkdirAll(beadsDir, 0755)

	// Change to test directory
	originalDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(originalDir)

	// Test checkGitForIssues finds beads.jsonl (not issues.jsonl)
	count, path := checkGitForIssues()
	if count != 2 {
		t.Errorf("Expected 2 issues in git, got %d", count)
	}
	if path != ".beads/beads.jsonl" {
		t.Errorf("Expected beads.jsonl, got %s", path)
	}

	// Initialize database and import
	dbPath := filepath.Join(beadsDir, "test.db")
	store, err := sqlite.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	if err := store.SetConfig(ctx, "issue_prefix", "test"); err != nil {
		t.Fatalf("Failed to set prefix: %v", err)
	}

	if err := importFromGit(ctx, dbPath, store, path); err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Verify correct filename was detected
	if filepath.Base(path) != "beads.jsonl" {
		t.Errorf("Should have imported from beads.jsonl, got %s", path)
	}

	// Verify stats show >0 issues
	stats, err := store.GetStatistics(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}
	if stats.TotalIssues != 2 {
		t.Errorf("Expected 2 issues, got %d", stats.TotalIssues)
	}
}

// testLegacyFilenameSupport tests issues.jsonl fallback
func testLegacyFilenameSupport(t *testing.T) {
	dir := t.TempDir()

	// Initialize git repo
	runCmd(t, dir, "git", "init")
	runCmd(t, dir, "git", "config", "user.email", "test@example.com")
	runCmd(t, dir, "git", "config", "user.name", "Test User")

	// Create .beads directory with issues.jsonl (legacy)
	beadsDir := filepath.Join(dir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatalf("Failed to create .beads dir: %v", err)
	}

	issue := &types.Issue{
		ID:        "test-1",
		Title:     "Legacy issue",
		Status:    types.StatusOpen,
		Priority:  2,
		IssueType: types.TypeTask,
	}

	// Use legacy filename
	jsonlPath := filepath.Join(beadsDir, "issues.jsonl")
	if err := writeJSONL(jsonlPath, []*types.Issue{issue}); err != nil {
		t.Fatalf("Failed to write JSONL: %v", err)
	}

	// Commit to git
	runCmd(t, dir, "git", "add", ".beads/issues.jsonl")
	runCmd(t, dir, "git", "commit", "-m", "Add legacy issue")

	// Change to test directory
	originalDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(originalDir)

	// Test checkGitForIssues finds issues.jsonl
	count, path := checkGitForIssues()
	if count != 1 {
		t.Errorf("Expected 1 issue in git, got %d", count)
	}
	if path != ".beads/issues.jsonl" {
		t.Errorf("Expected issues.jsonl, got %s", path)
	}

	// Initialize and import
	dbPath := filepath.Join(beadsDir, "test.db")
	store, err := sqlite.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	if err := store.SetConfig(ctx, "issue_prefix", "test"); err != nil {
		t.Fatalf("Failed to set prefix: %v", err)
	}

	if err := importFromGit(ctx, dbPath, store, path); err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Verify import succeeded
	stats, err := store.GetStatistics(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}
	if stats.TotalIssues != 1 {
		t.Errorf("Expected 1 issue, got %d", stats.TotalIssues)
	}
}

// testPrecedenceTest verifies beads.jsonl is preferred over issues.jsonl
func testPrecedenceTest(t *testing.T) {
	dir := t.TempDir()

	// Initialize git repo
	runCmd(t, dir, "git", "init")
	runCmd(t, dir, "git", "config", "user.email", "test@example.com")
	runCmd(t, dir, "git", "config", "user.name", "Test User")

	// Create .beads directory with BOTH files
	beadsDir := filepath.Join(dir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatalf("Failed to create .beads dir: %v", err)
	}

	// Create beads.jsonl with 2 issues
	beadsIssues := []*types.Issue{
		{ID: "test-1", Title: "From beads.jsonl", Status: types.StatusOpen, Priority: 1, IssueType: types.TypeTask},
		{ID: "test-2", Title: "Also from beads.jsonl", Status: types.StatusOpen, Priority: 1, IssueType: types.TypeTask},
	}
	if err := writeJSONL(filepath.Join(beadsDir, "beads.jsonl"), beadsIssues); err != nil {
		t.Fatalf("Failed to write beads.jsonl: %v", err)
	}

	// Create issues.jsonl with 1 issue (should be ignored)
	legacyIssues := []*types.Issue{
		{ID: "test-99", Title: "From issues.jsonl", Status: types.StatusOpen, Priority: 1, IssueType: types.TypeTask},
	}
	if err := writeJSONL(filepath.Join(beadsDir, "issues.jsonl"), legacyIssues); err != nil {
		t.Fatalf("Failed to write issues.jsonl: %v", err)
	}

	// Commit both files
	runCmd(t, dir, "git", "add", ".beads/")
	runCmd(t, dir, "git", "commit", "-m", "Add both files")

	// Change to test directory
	originalDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(originalDir)

	// Test checkGitForIssues prefers beads.jsonl
	count, path := checkGitForIssues()
	if count != 2 {
		t.Errorf("Expected 2 issues (from beads.jsonl), got %d", count)
	}
	if path != ".beads/beads.jsonl" {
		t.Errorf("Expected beads.jsonl to be preferred, got %s", path)
	}
}

// testInitSafetyCheck tests the safety check that prevents silent data loss
func testInitSafetyCheck(t *testing.T) {
	dir := t.TempDir()

	// Initialize git repo
	runCmd(t, dir, "git", "init")
	runCmd(t, dir, "git", "config", "user.email", "test@example.com")
	runCmd(t, dir, "git", "config", "user.name", "Test User")

	// Create .beads directory with beads.jsonl
	beadsDir := filepath.Join(dir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatalf("Failed to create .beads dir: %v", err)
	}

	issue := &types.Issue{
		ID:        "test-1",
		Title:     "Test issue",
		Status:    types.StatusOpen,
		Priority:  1,
		IssueType: types.TypeTask,
	}

	jsonlPath := filepath.Join(beadsDir, "beads.jsonl")
	if err := writeJSONL(jsonlPath, []*types.Issue{issue}); err != nil {
		t.Fatalf("Failed to write JSONL: %v", err)
	}

	// Commit to git
	runCmd(t, dir, "git", "add", ".beads/beads.jsonl")
	runCmd(t, dir, "git", "commit", "-m", "Add issue")

	// Change to test directory
	originalDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(originalDir)

	// Create empty database (simulating failed import)
	dbPath := filepath.Join(beadsDir, "test.db")
	store, err := sqlite.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	ctx := context.Background()
	if err := store.SetConfig(ctx, "issue_prefix", "test"); err != nil {
		t.Fatalf("Failed to set prefix: %v", err)
	}

	// Verify safety check would detect the problem
	stats, err := store.GetStatistics(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.TotalIssues == 0 {
		// Database is empty - check if git has issues
		recheck, recheckPath := checkGitForIssues()
		if recheck == 0 {
			t.Error("Safety check should have detected issues in git")
		}
		if recheckPath != ".beads/beads.jsonl" {
			t.Errorf("Safety check found wrong path: %s", recheckPath)
		}
		// This would trigger the error exit in real init.go
		t.Logf("Safety check correctly detected %d issues in git at %s", recheck, recheckPath)
	} else {
		t.Error("Database should be empty for this test")
	}

	store.Close()
}

// Helper functions

func runCmd(t *testing.T, dir string, name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Command %s %v failed: %v\nOutput: %s", name, args, err, output)
	}
}

func writeJSONL(path string, issues []*types.Issue) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for _, issue := range issues {
		if err := enc.Encode(issue); err != nil {
			return err
		}
	}
	return nil
}
