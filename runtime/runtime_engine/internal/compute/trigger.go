// Package compute contains deterministic RuntimeEngine V1 trigger decisions.
package compute

import (
	"bytes"
	"encoding/json"
	"io"
	"math/big"
	"time"

	"github.com/indu-forge/runtime-engine/internal/model"
)

type ChangeState struct {
	Value   json.RawMessage
	Quality string
	Seen    bool
}

// ChangeTriggered never converts JSON numbers through float64: point types
// include int64/uint64 and decimals, all of which must retain their exact
// wire value. Invalid or trailing JSON fails closed.
func ChangeTriggered(mode string, deadband json.RawMessage, previous ChangeState, current json.RawMessage, quality string) bool {
	if !strictJSON(current) || (previous.Seen && !strictJSON(previous.Value)) {
		return false
	}
	if !previous.Seen || mode == "any" || previous.Quality != quality {
		return true
	}
	if mode == "value_change" {
		if len(deadband) == 0 {
			return !jsonEqual(previous.Value, current)
		}
		return numericDeltaOver(previous.Value, current, deadband)
	}
	if mode == "rising_edge" || mode == "falling_edge" {
		before, beforeOK := boolJSON(previous.Value)
		after, afterOK := boolJSON(current)
		return beforeOK && afterOK && ((mode == "rising_edge" && !before && after) || (mode == "falling_edge" && before && !after))
	}
	before, beforeOK := ratJSON(previous.Value)
	after, afterOK := ratJSON(current)
	if !beforeOK || !afterOK {
		return false
	}
	delta := new(big.Rat).Sub(after, before)
	band, ok := ratJSON(deadband)
	if !ok {
		return false
	}
	switch mode {
	case "increase":
		return delta.Cmp(band) > 0
	case "decrease":
		return new(big.Rat).Neg(delta).Cmp(band) > 0
	}
	return false
}
func numericDeltaOver(before, after, deadband json.RawMessage) bool {
	a, aok := ratJSON(before)
	b, bok := ratJSON(after)
	band, bandOK := ratJSON(deadband)
	return aok && bok && bandOK && new(big.Rat).Abs(new(big.Rat).Sub(b, a)).Cmp(band) > 0
}
func IsAutomaticTrigger(trigger model.Trigger) bool {
	return trigger.Kind == "datapoint_change" || trigger.Kind == "condition"
}

func strictJSON(raw json.RawMessage) bool { _, ok := decodeStrict(raw); return ok }
func decodeStrict(raw json.RawMessage) (any, bool) {
	var v any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if d.Decode(&v) != nil {
		return nil, false
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		return nil, false
	}
	return v, true
}
func boolJSON(raw json.RawMessage) (bool, bool) {
	v, ok := decodeStrict(raw)
	b, is := v.(bool)
	return b, ok && is
}
func ratJSON(raw json.RawMessage) (*big.Rat, bool) {
	return model.ParseNonNegativeOrSignedNumber(raw)
}
func jsonEqual(left, right json.RawMessage) bool {
	// JSON spells of one numeric value (1, 1.0, 1e0) are equivalent for a
	// datapoint. Compare those exactly before falling back to structure.
	leftNumber, leftIsNumber := ratJSON(left)
	rightNumber, rightIsNumber := ratJSON(right)
	if leftIsNumber && rightIsNumber {
		return leftNumber.Cmp(rightNumber) == 0
	}
	a, aOK := decodeStrict(left)
	b, bOK := decodeStrict(right)
	if !aOK || !bOK {
		return false
	}
	return canonicalJSON(a) == canonicalJSON(b)
}
func canonicalJSON(value any) string { encoded, _ := json.Marshal(value); return string(encoded) }

func DebouncedPhase(active, previousActive bool, pending string, pendingSince *time.Time, now time.Time, debounce time.Duration) (phase, nextPending string, nextSince *time.Time) {
	desired := "exited"
	if active {
		desired = "entered"
	}
	if active == previousActive {
		if active && pending == "" {
			return "active", "", nil
		}
		return "", "", nil
	}
	if pending != desired || pendingSince == nil {
		at := now.UTC()
		return "", desired, &at
	}
	if now.UTC().Sub(pendingSince.UTC()) < debounce {
		return "", pending, pendingSince
	}
	return desired, "", nil
}
