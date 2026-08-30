package transportlimits

import (
	"errors"
	"strings"
	"testing"
)

const pointID = "11111111-1111-4111-8111-111111111111"

func TestOutboundPayloadLimitsAreExactAndSubjectsFailClosed(t *testing.T) {
	for _, test := range []struct {
		name, subject string
		limit         int
	}{
		{"raw", "data.raw." + pointID, MaxBodyBytes},
		{"computed", "data.computed." + pointID, MaxBodyBytes},
		{"alarm", "alarm.event", MaxBodyBytes},
		{"dlq", "dlq.compute-derived", MaxDLQPayloadBytes},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := ValidateOutboundPayload(test.subject, make([]byte, test.limit)); err != nil {
				t.Fatalf("exact boundary rejected: %v", err)
			}
			if err := ValidateOutboundPayload(test.subject, make([]byte, test.limit+1)); !errors.Is(err, ErrOutboundPayloadTooLarge) {
				t.Fatalf("over boundary error=%v", err)
			}
		})
	}
	for _, subject := range []string{"", "x", "data.", "data.raw", "data.raw.not-a-uuid", "data.computed.", "dlq.", "dlq.-writer", "dlq.writer?x", "dlq.writer/other"} {
		if err := ValidateOutboundPayload(subject, nil); !errors.Is(err, ErrOutboundSubject) {
			t.Fatalf("unknown/masquerading subject accepted: %q err=%v", subject, err)
		}
	}
	if err := ValidateOutboundPayload("dlq."+strings.Repeat("x", 129), nil); !errors.Is(err, ErrOutboundSubject) {
		t.Fatalf("oversized dlq consumer accepted: %v", err)
	}
}
