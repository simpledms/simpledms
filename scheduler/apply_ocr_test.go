package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/model"
	"github.com/simpledms/simpledms/util/ocrutil"
)

func TestApplyOCROneFileSkipsTooLargeFile(t *testing.T) {
	t.Setenv(ocrutil.MaxFileSizeMiBEnvVar, "1")

	now := time.Now()
	currentVersion := model.NewStoredFile(&enttenant.StoredFile{
		Size:                       (1024 * 1024) + 1,
		CopiedToFinalDestinationAt: &now,
	})

	content, fileNotReady, fileTooLarge, err := (&Scheduler{}).applyOCROneFile(
		context.Background(),
		nil,
		currentVersion,
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if fileNotReady {
		t.Fatalf("expected fileNotReady to be false")
	}
	if !fileTooLarge {
		t.Fatalf("expected fileTooLarge to be true")
	}
	if content != "" {
		t.Fatalf("expected empty OCR content for too-large file")
	}
}

func TestApplyOCROneFileReturnsNotReadyForUnmovedFile(t *testing.T) {
	t.Setenv(ocrutil.MaxFileSizeMiBEnvVar, "1")

	currentVersion := model.NewStoredFile(&enttenant.StoredFile{
		Size: 1024,
	})

	content, fileNotReady, fileTooLarge, err := (&Scheduler{}).applyOCROneFile(
		context.Background(),
		nil,
		currentVersion,
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !fileNotReady {
		t.Fatalf("expected fileNotReady to be true")
	}
	if fileTooLarge {
		t.Fatalf("expected fileTooLarge to be false")
	}
	if content != "" {
		t.Fatalf("expected empty OCR content for not-ready file")
	}
}
