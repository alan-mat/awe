package engine_test

import (
	"errors"
	"testing"
	"time"

	"github.com/alan-mat/awe/internal/engine"
)

func TestTraceComplete(t *testing.T) {
	trace := engine.NewTrace("test-id", engine.TextQuery("My query"), engine.CallerMeta{Name: "test-user"})
	now := time.Now().UnixNano()

	// complete fresh trace
	trace.Complete()
	if trace.CompletedAt < now {
		t.Errorf("invalid trace completed at time, expected '%v' or later, got '%v'", now, trace.CompletedAt)
	}
	completedAt := trace.CompletedAt
	if trace.Status != engine.TraceStatusCompleted {
		t.Error("invalid trace status after call to Complete()")
	}

	// calling complete on an already completed trace
	trace.Complete()
	if trace.CompletedAt != completedAt {
		t.Error("Complete() call on already completed trace overrid the completion time")
	}
	if trace.Status != engine.TraceStatusCompleted {
		t.Error("invalid trace status after call to Complete()")
	}

	// may not fail completed trace
	reason := "trace failed!"
	trace.Fail(errors.New(reason))
	if trace.Status != engine.TraceStatusCompleted {
		t.Error("invalid trace status after failing a completed trace")
	}
	if trace.FailReason != nil {
		t.Error("fail reason set on already completed trace")
	}
	if trace.CompletedAt != completedAt {
		t.Error("Fail() call on already completed trace overrid the completion time")
	}
}

func TestTraceFail(t *testing.T) {
	trace := engine.NewTrace("test-id", engine.TextQuery("My query"), engine.CallerMeta{Name: "test-user"})
	now := time.Now().UnixNano()
	reason := "trace failed!"

	// fail fresh trace
	trace.Fail(errors.New(reason))
	if trace.CompletedAt < now {
		t.Errorf("invalid trace completed at time, expected '%v' or later, got '%v'", now, trace.CompletedAt)
	}
	completedAt := trace.CompletedAt
	if trace.Status != engine.TraceStatusFailed {
		t.Error("invalid trace status after call to Fail()")
	}
	if *trace.FailReason != reason {
		t.Errorf("invalid trace fail reason, expected '%s', got '%s'", reason, *trace.FailReason)
	}

	// calling fail on an already failed trace
	trace.Fail(errors.New(reason))
	if trace.CompletedAt != completedAt {
		t.Error("Fail() call on already completed trace overrid the completion time")
	}
	if trace.Status != engine.TraceStatusFailed {
		t.Error("invalid trace status after call to Fail()")
	}
	if *trace.FailReason != reason {
		t.Errorf("invalid trace fail reason, expected '%s', got '%s'", reason, *trace.FailReason)
	}

	// may not complete failed trace
	trace.Complete()
	if trace.CompletedAt != completedAt {
		t.Error("Complete() call on already failed trace overrid the completion time")
	}
	if trace.Status != engine.TraceStatusFailed {
		t.Error("invalid trace status after calling complete on already failed trace")
	}
	if *trace.FailReason != reason {
		t.Errorf("invalid trace fail reason, expected '%s', got '%s'", reason, *trace.FailReason)
	}
}
