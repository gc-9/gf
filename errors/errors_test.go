package errors

import (
	stdErrors "errors"
	"fmt"
	"strings"
	"testing"
)

func TestWrapCapturesOnlyOneStack(t *testing.T) {
	cause := stdErrors.New("database unavailable")
	first := Wrap(cause, "create admin")
	second := Wrap(first, "store admin")

	firstErr := first.(*ErrMessage)
	secondErr := second.(*ErrMessage)
	if firstErr.Stack == nil {
		t.Fatal("first wrap should capture a stack trace")
	}
	if frame := firstErr.StackTrace()[0]; !strings.Contains(frame.name(), "TestWrapCapturesOnlyOneStack") {
		t.Fatalf("stack should start at the wrapping call site, got %s", frame.name())
	}
	if secondErr.Stack != nil {
		t.Fatal("nested wrap should reuse the existing stack trace")
	}
	if !stdErrors.Is(second, cause) {
		t.Fatal("wrapped cause should be discoverable with errors.Is")
	}
}

func TestPublicOptionsAndWrap(t *testing.T) {
	public := Public("notFound", WithCode(404), WithInternalCode(1001)).(*ErrMessage)
	if !public.Public || public.Message != "notFound" || public.Code != 404 || public.InternalCode != 1001 {
		t.Fatalf("unexpected public error: %#v", public)
	}

	wrapped := PublicWrap(stdErrors.New("write failed"), "文件保存失败", WithInternalCode(2001)).(*ErrMessage)
	if !wrapped.Public || wrapped.Cause == nil || wrapped.Stack == nil || wrapped.InternalCode != 2001 {
		t.Fatalf("unexpected public wrapped error: %#v", wrapped)
	}
	if got := fmt.Sprintf("%+v", Public("safe message")); got != "safe message" {
		t.Fatalf("formatting an error without a stack failed: %q", got)
	}
}

func TestEnsurePublic(t *testing.T) {
	t.Run("preserves an existing public error", func(t *testing.T) {
		original := Public("notFound")
		if got := EnsurePublic(original, "操作失败"); got != original {
			t.Fatal("public error should not be wrapped again")
		}
	})

	t.Run("wraps an internal error", func(t *testing.T) {
		original := stdErrors.New("database unavailable")
		wrapped := EnsurePublic(original, "dbError").(*ErrMessage)
		if !wrapped.Public || wrapped.Message != "dbError" {
			t.Fatalf("unexpected public error: %#v", wrapped)
		}
		if !stdErrors.Is(wrapped, original) {
			t.Fatal("wrapped error should retain its cause")
		}
	})
}

func TestStackDepth(t *testing.T) {
	previous := stackDepth
	t.Cleanup(func() { SetStackDepth(previous) })

	SetStackDepth(1)
	if got := stackDepth; got != 1 {
		t.Fatalf("got stack depth %d, want 1", got)
	}
	if got := len(callers(1).StackTrace()); got != 1 {
		t.Fatalf("got stack frames %d, want 1", got)
	}

	t.Run("rejects non-positive depth", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("SetStackDepth should panic for a non-positive depth")
			}
		}()
		SetStackDepth(0)
	})
}
