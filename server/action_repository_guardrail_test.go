package server

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestActionPackageDoesNotUseLegacyInfraFileRepo(t *testing.T) {
	t.Helper()

	repoRoot := repoRootPath(t)
	actionRoot := filepath.Join(repoRoot, "action")

	violations := make([]string, 0)
	err := filepath.WalkDir(actionRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if strings.Contains(string(content), "infra.FileRepo") {
			relPath, err := filepath.Rel(repoRoot, path)
			if err != nil {
				relPath = path
			}
			violations = append(violations, relPath)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walk action package: %v", err)
	}

	if len(violations) > 0 {
		t.Fatalf(
			"legacy infra.FileRepo usage is forbidden in action package; violations: %s",
			strings.Join(violations, ", "),
		)
	}
}

func TestCodebaseDoesNotReferenceRemovedFileRepositoryAdapter(t *testing.T) {
	t.Helper()

	repoRoot := repoRootPath(t)
	patterns := []string{"infra.FileRepo", "NewFileRepository("}
	violations := make([]string, 0)

	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			base := filepath.Base(path)
			if base == ".git" || base == "node_modules" {
				return fs.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		if filepath.Base(path) == "action_repository_guardrail_test.go" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		for _, pattern := range patterns {
			if !strings.Contains(string(content), pattern) {
				continue
			}
			relPath, err := filepath.Rel(repoRoot, path)
			if err != nil {
				relPath = path
			}
			violations = append(violations, relPath+": "+pattern)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walk repository: %v", err)
	}

	if len(violations) > 0 {
		t.Fatalf(
			"removed legacy file repository adapter references found: %s",
			strings.Join(violations, ", "),
		)
	}
}

func TestActionPackageDoesNotUseDirectFileUpdateQueries(t *testing.T) {
	t.Helper()

	repoRoot := repoRootPath(t)
	actionRoot := filepath.Join(repoRoot, "action")
	patterns := []string{"TTx.File.Update(", "TTx.File.UpdateOneID("}
	violations := make([]string, 0)

	err := filepath.WalkDir(actionRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		for _, pattern := range patterns {
			if !strings.Contains(string(content), pattern) {
				continue
			}

			relPath, err := filepath.Rel(repoRoot, path)
			if err != nil {
				relPath = path
			}
			violations = append(violations, relPath+": "+pattern)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walk action package: %v", err)
	}

	if len(violations) > 0 {
		t.Fatalf(
			"direct TTx.File.Update* usage is forbidden in action package; violations: %s",
			strings.Join(violations, ", "),
		)
	}
}

func TestActionPackageDoesNotUseDirectFileGetXQueries(t *testing.T) {
	t.Helper()

	repoRoot := repoRootPath(t)
	actionRoot := filepath.Join(repoRoot, "action")
	pattern := "TTx.File.GetX("
	violations := make([]string, 0)

	err := filepath.WalkDir(actionRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if !strings.Contains(string(content), pattern) {
			return nil
		}

		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			relPath = path
		}
		violations = append(violations, relPath+": "+pattern)

		return nil
	})
	if err != nil {
		t.Fatalf("walk action package: %v", err)
	}

	if len(violations) > 0 {
		t.Fatalf(
			"direct TTx.File.GetX usage is forbidden in action package; violations: %s",
			strings.Join(violations, ", "),
		)
	}
}

func TestActionPackageDoesNotUseFilesystemFileModelHydrationHelpers(t *testing.T) {
	t.Helper()

	repoRoot := repoRootPath(t)
	actionRoot := filepath.Join(repoRoot, "action")
	patterns := []string{"FileModelByPublicIDX(", "FileModelByPublicIDWithDeletedX("}
	violations := make([]string, 0)

	err := filepath.WalkDir(actionRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		for _, pattern := range patterns {
			if !strings.Contains(string(content), pattern) {
				continue
			}

			relPath, err := filepath.Rel(repoRoot, path)
			if err != nil {
				relPath = path
			}
			violations = append(violations, relPath+": "+pattern)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walk action package: %v", err)
	}

	if len(violations) > 0 {
		t.Fatalf(
			"filesystem FileModelBy* hydration helper usage is forbidden in action package; violations: %s",
			strings.Join(violations, ", "),
		)
	}
}

func TestCodebaseDoesNotUseDeprecatedFileSystemMoveRenameAPIs(t *testing.T) {
	t.Helper()

	repoRoot := repoRootPath(t)
	patterns := []string{".FileSystem().Move(", ".FileSystem().Rename("}
	violations := make([]string, 0)

	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			base := filepath.Base(path)
			if base == ".git" || base == "node_modules" {
				return fs.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		if filepath.Base(path) == "action_repository_guardrail_test.go" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		for _, pattern := range patterns {
			if !strings.Contains(string(content), pattern) {
				continue
			}

			relPath, err := filepath.Rel(repoRoot, path)
			if err != nil {
				relPath = path
			}
			violations = append(violations, relPath+": "+pattern)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walk repository: %v", err)
	}

	if len(violations) > 0 {
		t.Fatalf(
			"deprecated FileSystem Move/Rename API usage found: %s",
			strings.Join(violations, ", "),
		)
	}
}

func repoRootPath(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}

	return filepath.Dir(filepath.Dir(thisFile))
}
