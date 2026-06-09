package selfupdate

import "testing"

func TestCleanEnvStripsLoaderVars(t *testing.T) {
	orig := osEnviron
	t.Cleanup(func() { osEnviron = orig })
	osEnviron = func() []string {
		return []string{
			"LD_PRELOAD=/x",
			"LD_LIBRARY_PATH=/y",
			"PGVOYAGER_PORT=1",
			"FOO=bar",
			"DYLD_INSERT_LIBRARIES=/z",
			"DYLD_LIBRARY_PATH=/w",
		}
	}

	got := cleanEnv()

	// Build a set for easy lookup.
	present := make(map[string]bool, len(got))
	for _, kv := range got {
		present[kv] = true
	}

	// These must be dropped.
	for _, bad := range []string{
		"LD_PRELOAD=/x",
		"LD_LIBRARY_PATH=/y",
		"PGVOYAGER_PORT=1",
		"DYLD_INSERT_LIBRARIES=/z",
		"DYLD_LIBRARY_PATH=/w",
	} {
		if present[bad] {
			t.Errorf("cleanEnv() kept %q, want it dropped", bad)
		}
	}

	// FOO=bar must be kept.
	if !present["FOO=bar"] {
		t.Errorf("cleanEnv() dropped FOO=bar, want it kept")
	}
}
