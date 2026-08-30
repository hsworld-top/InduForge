package eventid

import "testing"

func TestContractTruthVectors(t *testing.T) {
	raw, err := Raw("data.raw.v1", "deployment-line1-prod", "dp-line1-temperature", "collector-line1-a", 7, 1024)
	if err != nil || raw != "c3e246a31546e09d6f8e7b716235574d1f2cba2499e70fc39c70f2e60b124371" {
		t.Fatalf("raw=%q err=%v", raw, err)
	}
	computed, err := Computed("data.computed.v1", "deployment-line1-prod", "dp-line1-temperature-fahrenheit", "runtime-engine-compute-0", 7, 1024, "compute-line1-fahrenheit", 3, []string{"c3e246a31546e09d6f8e7b716235574d1f2cba2499e70fc39c70f2e60b124371"})
	if err != nil || computed != "63409b7a499ea1c4ace7be3d5e73b4fded56d433cab0cac1f1a207660dd35de6" {
		t.Fatalf("computed=%q err=%v", computed, err)
	}
	alarm, err := AlarmTransition("alarm.event.v1", "alarm-transition", "deployment-line1-prod", "alarm-line1-temperature-high-20260830", "RAISE", "runtime-engine-alarm-0", 7, "2026-08-30T10:20:30.123Z", "alarm-item-line1-temperature-high", []string{"dp-line1-temperature"}, 1, 1)
	if err != nil || alarm != "c7bf4d41a3701792a20944428640f328c0b098349fa65e764d2353c16a2cffdb" {
		t.Fatalf("alarm=%q err=%v", alarm, err)
	}
	gap, err := DataGap("alarm.event.v1", "data-gap", "deployment-line1-prod", "collector-line1-a", "line1-plc", "collector-line1-a", 7, 1000, 1024, "2026-08-30T10:20:31Z", "wal-capacity")
	if err != nil || gap != "9d37f055c4d1a69b0bb84209eaf81c3e728114ec4c83ccfeb84774e99cbaf33b" {
		t.Fatalf("gap=%q err=%v", gap, err)
	}
}

func TestHashFieldsRejectsSeparatorAndBodyUsesOriginalBytes(t *testing.T) {
	if _, err := HashFields("a\x1fb"); err == nil {
		t.Fatal("expected separator rejection")
	}
	if BodySHA256([]byte(`{"a":1}`)) == BodySHA256([]byte(`{ "a": 1 }`)) {
		t.Fatal("body hash must retain original bytes")
	}
}
