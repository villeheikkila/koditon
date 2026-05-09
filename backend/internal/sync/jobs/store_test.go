package syncjobs

import "testing"

func TestCanonicalJobsUseCanonicalQueue(t *testing.T) {
	t.Parallel()
	if got := QueueNameForProvider("canonical"); got != "canonical" {
		t.Fatalf("QueueNameForProvider(canonical) = %q", got)
	}
	def, ok := FindJobDefinition("canonical", "canonical_rebuild_dimension_layer_backfill")
	if !ok {
		t.Fatal("canonical dimension layer backfill definition missing")
	}
	if def.Queue != "canonical" {
		t.Fatalf("definition queue = %q", def.Queue)
	}
	if def.CapacityClass != CapacityClassInternalDB {
		t.Fatalf("definition capacity class = %q", def.CapacityClass)
	}
	if def.MaxInProgress != 1 {
		t.Fatalf("definition max in progress = %d", def.MaxInProgress)
	}
	if _, ok := FindJobDefinition("canonical", "canonical_rebuild_dimension_layer_listing"); !ok {
		t.Fatal("canonical dimension layer listing definition missing")
	}
	if _, ok := FindJobDefinition("canonical", "canonical_resolve_dirty_dimension_targets"); !ok {
		t.Fatal("canonical dirty dimension targets definition missing")
	}
	if _, ok := FindJobDefinition("canonical", "canonical_resolve_dimension_target"); !ok {
		t.Fatal("canonical dimension target definition missing")
	}
}

func TestExecutionPolicyUsesJobDefinitions(t *testing.T) {
	t.Parallel()
	policy := DefaultExecutionPolicy()
	for _, def := range jobDefinitions {
		if got := policy.KindMaxInProgress[def.Kind]; got != def.MaxInProgress {
			t.Fatalf("KindMaxInProgress[%s] = %d, want %d", def.Kind, got, def.MaxInProgress)
		}
	}
}
