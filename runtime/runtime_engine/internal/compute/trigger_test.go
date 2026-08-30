package compute

import "testing"

func TestChangeTriggeredModes(t *testing.T) {
	previous := ChangeState{Value: []byte(`10`), Quality: "good", Seen: true}
	if !ChangeTriggered("increase", []byte(`2`), previous, []byte(`13`), "good") || ChangeTriggered("increase", []byte(`3`), previous, []byte(`13`), "good") {
		t.Fatal("increase/deadband")
	}
	if !ChangeTriggered("decrease", []byte(`2`), previous, []byte(`7`), "good") || ChangeTriggered("increase", []byte(`0`), previous, []byte(`9`), "good") {
		t.Fatal("direction modes")
	}
	if !ChangeTriggered("rising_edge", nil, ChangeState{Value: []byte(`false`), Quality: "good", Seen: true}, []byte(`true`), "good") {
		t.Fatal("rising edge")
	}
	if !ChangeTriggered("value_change", nil, previous, []byte(`11`), "good") || ChangeTriggered("value_change", nil, previous, []byte(`10`), "good") {
		t.Fatal("value change")
	}
}

func TestChangeTriggeredPreservesLargeIntegersAndDecimals(t *testing.T) {
	for _, pair := range [][2]string{{"9007199254740992", "9007199254740993"}, {"18446744073709551614", "18446744073709551615"}, {"1.0000000000000000000000000001", "1.0000000000000000000000000002"}} {
		previous := ChangeState{Value: []byte(pair[0]), Quality: "good", Seen: true}
		if !ChangeTriggered("value_change", nil, previous, []byte(pair[1]), "good") || !ChangeTriggered("increase", []byte(`0`), previous, []byte(pair[1]), "good") {
			t.Fatalf("精确数值变化漏报: %v", pair)
		}
	}
	previous := ChangeState{Value: []byte(`9007199254740992`), Quality: "good", Seen: true}
	if ChangeTriggered("value_change", nil, ChangeState{Value: []byte(`1`), Quality: "good", Seen: true}, []byte(`1.0`), "good") ||
		ChangeTriggered("value_change", nil, ChangeState{Value: []byte(`1.0`), Quality: "good", Seen: true}, []byte(`1e0`), "good") {
		t.Fatal("equivalent numeric spellings must not trigger")
	}
	if !ChangeTriggered("value_change", nil, previous, []byte(`9007199254740993`), "good") {
		t.Fatal("adjacent large integers must still trigger")
	}
	if ChangeTriggered("value_change", nil, previous, []byte(`9007199254740992 trailing`), "good") {
		t.Fatal("trailing JSON 不得触发")
	}
	previous = ChangeState{Value: []byte(`1.0000000000000000000000000001`), Quality: "good", Seen: true}
	if !ChangeTriggered("value_change", []byte(`0.00000000000000000000000000005`), previous, []byte(`1.0000000000000000000000000002`), "good") {
		t.Fatal("high precision deadband lost a change")
	}
	if ChangeTriggered("value_change", []byte(`0.0000000000000000000000000001`), previous, []byte(`1.0000000000000000000000000002`), "good") {
		t.Fatal("deadband equality must not trigger")
	}
}
