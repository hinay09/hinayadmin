package job

import (
	"testing"

	"hinay.cn/admin/internal/consts"
)

func TestLockKey(t *testing.T) {
	got := lockKey(2, 37)
	if got != consts.JobLockPrefix+"2:37" {
		t.Fatalf("lockKey = %q", got)
	}
}

func TestNormalizeConcurrentPolicy(t *testing.T) {
	for in, want := range map[string]string{
		"allow": consts.JobConcurrentAllow,
		"skip":  consts.JobConcurrentSkip,
		"":      consts.JobConcurrentAllow,
		"other": consts.JobConcurrentAllow,
	} {
		if got := normalizeConcurrentPolicy(in); got != want {
			t.Fatalf("normalizeConcurrentPolicy(%q) = %q, want %q", in, got, want)
		}
	}
}
