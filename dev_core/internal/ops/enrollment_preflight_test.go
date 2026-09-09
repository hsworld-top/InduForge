package ops

import "testing"

func TestEnrollmentCapabilitiesAreRequirements(t *testing.T) {
	full := ClaimEnrollmentInput{Platform: "linux", Capabilities: []string{"project_entry", "data_runtime", "collector"}}
	if !enrollmentMatches(full, "linux", []string{"project_entry", "data_runtime"}) {
		t.Fatal("additional installed collector must not reject standard node")
	}
	if enrollmentMatches(full, "windows", []string{"collector"}) {
		t.Fatal("wrong platform accepted")
	}
	if enrollmentMatches(ClaimEnrollmentInput{Platform: "linux", Capabilities: []string{"collector"}}, "linux", []string{"project_entry", "data_runtime"}) {
		t.Fatal("missing required runtime accepted")
	}
}
