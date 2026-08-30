package alarm

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/indu-forge/runtime-engine/internal/model"
)

const pointID = "22222222-2222-4222-8222-222222222222"
const itemID = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
const conditionID = "aaaaaaa1-aaaa-4aaa-8aaa-aaaaaaaaaaaa"

func TestRaiseSeverityClearAndRoundTrip(t *testing.T) {
	critical := model.AlarmCondition{ID: "bbbbbbb1-bbbb-4bbb-8bbb-bbbbbbbbbbbb", Kind: "threshold", Operator: "gt", Params: json.RawMessage(`{"threshold":20}`), Severity: "critical", Deadband: json.RawMessage(`0`)}
	r := newRuntime(t, []model.AlarmCondition{condition("gt", 10, "warning", 0, 0), critical})
	base := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	if got, err := r.Apply(input("a", 11, 1, base)); err != nil || len(got) != 1 || got[0].Operation != "RAISE" || got[0].Transition.AlarmState != "OPEN" || got[0].Transition.PreviousSeverity != nil {
		t.Fatalf("raise: %#v %v", got, err)
	}
	if got, err := r.Apply(input("b", 21, 2, base.Add(time.Second))); err != nil || len(got) != 1 || got[0].Operation != "SEVERITY_CHANGE" || got[0].Transition.PreviousSeverity == nil || *got[0].Transition.PreviousSeverity != "warning" {
		t.Fatalf("severity: %#v %v", got, err)
	}
	state := r.State()
	raw, _ := json.Marshal(state)
	var restored State
	if err := json.Unmarshal(raw, &restored); err != nil {
		t.Fatal(err)
	}
	r, _ = NewRuntime(identity(), artifact([]model.AlarmCondition{condition("gt", 10, "warning", 0, 0), critical}), restored)
	if got, err := r.Apply(input("c", 1, 3, base.Add(2*time.Second))); err != nil || len(got) != 1 || got[0].Operation != "CLEAR" || got[0].Transition.ClearedAt == nil || got[0].Transition.AckedAt != nil {
		t.Fatalf("clear: %#v %v", got, err)
	}
}

func TestDelayOutOfOrderAndSweepStale(t *testing.T) {
	r := newRuntime(t, []model.AlarmCondition{condition("gt", 10, "warning", 1000, 1000)})
	base := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	if got, _ := r.Apply(input("a", 11, 2, base)); len(got) != 0 {
		t.Fatal("trigger delay must hold")
	}
	if got, _ := r.Apply(input("b", 100, 1, base.Add(-time.Second))); len(got) != 0 {
		t.Fatal("out of order input must not advance")
	}
	if got, _ := r.Sweep(base.Add(time.Second)); len(got) != 1 || got[0].Operation != "RAISE" {
		t.Fatalf("sweep trigger=%#v", got)
	}
	if got, _ := r.Apply(input("b", 1, 3, base.Add(2*time.Second))); len(got) != 0 {
		t.Fatal("clear delay must hold")
	}
	if got, _ := r.Sweep(base.Add(3 * time.Second)); len(got) != 1 || got[0].Operation != "CLEAR" {
		t.Fatalf("sweep clear=%#v", got)
	}

	stale := model.AlarmCondition{ID: conditionID, Kind: "stale", Operator: "age_gte", Params: json.RawMessage(`{"maxAgeMs":1000}`), Severity: "warning"}
	r = newRuntime(t, []model.AlarmCondition{stale})
	if got, err := r.Apply(input("d", 1, 1, base)); err != nil || len(got) != 0 {
		t.Fatalf("initial stale=%#v %v", got, err)
	}
	if got, err := r.Sweep(base.Add(time.Second)); err != nil || len(got) != 1 || got[0].Operation != "RAISE" {
		t.Fatalf("stale sweep=%#v %v", got, err)
	}
	if due := NextEvaluationAt(artifact([]model.AlarmCondition{stale}).AlarmItems[0], r.State().Items[itemID], base.Add(time.Second)); due != nil {
		t.Fatalf("active stale alarm must wait for a new input, got due=%s", due)
	}
}

func TestNewerSequenceWithEarlierServerTimeStillEvaluates(t *testing.T) {
	base := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	r := newRuntime(t, []model.AlarmCondition{condition("gt", 10, "warning", 0, 0)})
	first := input("a", 0, 1, base)
	first.ServerTimestamp = base.Add(time.Minute)
	if got, err := r.Apply(first); err != nil || len(got) != 0 {
		t.Fatalf("first=%#v %v", got, err)
	}
	second := input("b", 20, 2, base.Add(30*time.Second))
	second.ServerTimestamp = base.Add(45 * time.Second)
	if got, err := r.Apply(second); err != nil || len(got) != 1 || got[0].Operation != "RAISE" {
		t.Fatalf("newer sequence with earlier server time=%#v %v", got, err)
	} else if got[0].ServerTimestamp != base.Add(time.Minute).Format(time.RFC3339Nano) || got[0].SourceTimestamp != second.SourceTimestamp.Format(time.RFC3339Nano) {
		t.Fatalf("event timestamps regressed: %#v", got[0])
	}
}

func TestRevisionResetPreservesTransitionCountersForSameTimestamp(t *testing.T) {
	base := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	item := artifact([]model.AlarmCondition{condition("gt", 10, "warning", 0, 0)}).AlarmItems[0]
	r, err := NewRuntime(identity(), model.ProjectArtifact{AlarmItems: []model.AlarmItem{item}}, State{})
	if err != nil {
		t.Fatal(err)
	}
	raise, err := r.Apply(input("a", 11, 1, base))
	if err != nil || len(raise) != 1 {
		t.Fatalf("old raise=%#v %v", raise, err)
	}
	if _, err = r.Apply(input("b", 1, 2, base.Add(time.Second))); err != nil {
		t.Fatal(err)
	}
	old := r.State().Items[itemID]
	if old.ActiveConditionID != "" || old.StateVersion != old.TransitionSequence || old.StateVersion < 2 {
		t.Fatalf("old inactive state=%#v", old)
	}
	item.Revision = 2
	reset := ItemState{Inputs: map[string]SampleState{}, StateVersion: old.StateVersion, TransitionSequence: old.TransitionSequence}
	r, err = NewRuntime(identity(), model.ProjectArtifact{AlarmItems: []model.AlarmItem{item}}, State{Items: map[string]ItemState{itemID: reset}})
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := r.Apply(input("c", 11, 3, base))
	if err != nil || len(fresh) != 1 || fresh[0].Operation != "RAISE" {
		t.Fatalf("revision raise=%#v %v", fresh, err)
	}
	if fresh[0].EventID == raise[0].EventID || fresh[0].AlarmID == raise[0].AlarmID || fresh[0].Transition.StateVersion <= old.StateVersion {
		t.Fatalf("revision reset reused identity/counter: old=%#v new=%#v", raise[0], fresh[0])
	}
}

func TestExpressionAndFiniteValueFailClosed(t *testing.T) {
	item := model.AlarmItem{ID: itemID, Revision: 1, DisplayName: "derived", Enabled: true, Mode: "derived", EvaluationMode: "single", Inputs: []model.Input{{Alias: "a", DatapointID: pointID}, {Alias: "b", DatapointID: "33333333-3333-4333-8333-333333333333"}}, DerivedExpression: ptr("a + b"), Conditions: []model.AlarmCondition{condition("gt", 10, "warning", 0, 0)}}
	if _, err := NewRuntime(identity(), model.ProjectArtifact{AlarmItems: []model.AlarmItem{item}}, State{}); err != nil {
		t.Fatal(err)
	}
	if _, err := evalExpression("f()", map[string]json.RawMessage{}); err == nil {
		t.Fatal("function call must be rejected")
	}
	if _, err := evalExpression("1 / 0", map[string]json.RawMessage{}); err == nil {
		t.Fatal("non-finite result must be rejected")
	}
}

func TestDerivedPausedInputRoundTripsAndRecovers(t *testing.T) {
	base := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	otherID := "33333333-3333-4333-8333-333333333333"
	for _, pausedQuality := range []string{"bad", "unknown"} {
		t.Run(pausedQuality, func(t *testing.T) {
			item := model.AlarmItem{ID: itemID, Revision: 1, DisplayName: "derived", Enabled: true, Mode: "derived", EvaluationMode: "single", Inputs: []model.Input{{Alias: "a", DatapointID: pointID}, {Alias: "b", DatapointID: otherID}}, DerivedExpression: ptr("a + b"), Conditions: []model.AlarmCondition{condition("gt", 10, "warning", 0, 0)}}
			r, err := NewRuntime(identity(), model.ProjectArtifact{AlarmItems: []model.AlarmItem{item}}, State{})
			if err != nil {
				t.Fatal(err)
			}
			if got, err := r.Apply(input("a", 7, 1, base)); err != nil || len(got) != 0 {
				t.Fatalf("first derived input=%#v %v", got, err)
			}
			paused := input("b", 7, 2, base.Add(time.Second))
			paused.PointID, paused.Quality = otherID, pausedQuality
			if got, err := r.Apply(paused); err != nil || len(got) != 0 {
				t.Fatalf("paused derived input=%#v %v", got, err)
			}
			state := r.State().Items[itemID]
			if len(state.LastValue) != 0 || state.LastQuality != "" || state.LastSourceAt != nil || state.LastEvaluatedAt != nil {
				t.Fatalf("paused inactive must retain no last state: %#v", state)
			}
			raw, _ := json.Marshal(r.State())
			var restored State
			if err := json.Unmarshal(raw, &restored); err != nil {
				t.Fatal(err)
			}
			r, err = NewRuntime(identity(), model.ProjectArtifact{AlarmItems: []model.AlarmItem{item}}, restored)
			if err != nil {
				t.Fatalf("persisted paused derived state rejected: %v", err)
			}
			good := input("c", 7, 3, base.Add(2*time.Second))
			good.PointID = otherID
			if got, err := r.Apply(good); err != nil || len(got) != 1 || got[0].Operation != "RAISE" {
				t.Fatalf("good derived recovery=%#v %v", got, err)
			}
		})
	}
}

func TestRateAndTransition(t *testing.T) {
	base := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	rate := model.AlarmCondition{ID: conditionID, Kind: "rate_of_change", Operator: "gt", Params: json.RawMessage(`{"direction":"rise","limit":5,"windowMs":60000}`), Severity: "warning"}
	r := newRuntime(t, []model.AlarmCondition{rate})
	r.Apply(input("a", 0, 1, base))
	if got, err := r.Apply(input("b", 10, 2, base.Add(time.Second))); err != nil || len(got) != 1 || got[0].Operation != "RAISE" {
		t.Fatalf("rate=%#v %v", got, err)
	}
	transition := model.AlarmCondition{ID: conditionID, Kind: "transition", Operator: "rising", Params: json.RawMessage(`{}`), Severity: "warning"}
	r = newRuntime(t, []model.AlarmCondition{transition})
	r.Apply(boolInput("a", false, 1, base))
	if got, err := r.Apply(boolInput("b", true, 2, base.Add(time.Second))); err != nil || len(got) != 1 || got[0].Operation != "RAISE" {
		t.Fatalf("transition=%#v %v", got, err)
	}
}

func TestOfflineIsExplicitAndParamsRejectTrailingJSON(t *testing.T) {
	base := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	offline := model.AlarmCondition{ID: conditionID, Kind: "offline", Operator: "is", Params: json.RawMessage(`{}`), Severity: "warning"}
	r := newRuntime(t, []model.AlarmCondition{offline})
	if got, err := r.Apply(input("a", 1, 1, base)); err != nil || len(got) != 0 {
		t.Fatalf("offline cannot infer from time: %#v %v", got, err)
	}
	p := input("b", 1, 2, base.Add(time.Second))
	p.Offline = true
	if got, err := r.Apply(p); err != nil || len(got) != 1 || got[0].Operation != "RAISE" {
		t.Fatalf("explicit offline=%#v %v", got, err)
	}
	bad := condition("gt", 1, "warning", 0, 0)
	bad.Params = json.RawMessage(`{"threshold":1}{}`)
	if _, err := NewRuntime(identity(), artifact([]model.AlarmCondition{bad}), State{}); err == nil {
		t.Fatal("trailing params JSON must fail closed")
	}
}

func TestBadOrOfflineNeverClearActiveAlarm(t *testing.T) {
	base := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	r := newRuntime(t, []model.AlarmCondition{condition("gt", 10, "warning", 0, 1000)})
	if got, err := r.Apply(input("a", 11, 1, base)); err != nil || len(got) != 1 {
		t.Fatalf("raise=%#v %v", got, err)
	}
	bad := input("b", 1, 2, base.Add(time.Second))
	bad.Quality = "bad"
	if got, err := r.Apply(bad); err != nil || len(got) != 0 {
		t.Fatalf("bad input must pause=%#v %v", got, err)
	}
	if got, err := r.Sweep(base.Add(24 * time.Hour)); err != nil || len(got) != 0 || r.State().Items[itemID].ActiveConditionID == "" {
		t.Fatalf("bad pause cleared active: %#v %v %#v", got, err, r.State())
	}
	offline := input("c", 1, 3, base.Add(24*time.Hour))
	offline.Offline = true
	if got, err := r.Apply(offline); err != nil || len(got) != 0 {
		t.Fatalf("offline input must pause=%#v %v", got, err)
	}
	if got, err := r.Sweep(base.Add(48 * time.Hour)); err != nil || len(got) != 0 || r.State().Items[itemID].ActiveConditionID == "" {
		t.Fatalf("offline pause cleared active: %#v %v %#v", got, err, r.State())
	}
	if got, err := r.Apply(input("d", 11, 4, base.Add(48*time.Hour))); err != nil || len(got) != 0 || r.State().Items[itemID].ActiveConditionID == "" {
		t.Fatalf("good recovery must preserve active=%#v %v", got, err)
	}
}

func TestStableEventOrderAndAtomicApply(t *testing.T) {
	base := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	first := model.AlarmItem{ID: "11111111-1111-4111-8111-111111111111", Revision: 1, DisplayName: "first", Enabled: true, Mode: "point", EvaluationMode: "single", Inputs: []model.Input{{Alias: "v", DatapointID: pointID}}, Conditions: []model.AlarmCondition{condition("gt", 10, "warning", 0, 0)}}
	secondCondition := condition("gt", 10, "warning", 0, 0)
	secondCondition.ID = "22222221-2222-4222-8222-222222222222"
	second := first
	second.ID = "22222222-2222-4222-8222-222222222222"
	second.DisplayName = "second"
	second.Conditions = []model.AlarmCondition{secondCondition}
	r, err := NewRuntime(identity(), model.ProjectArtifact{AlarmItems: []model.AlarmItem{second, first}}, State{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.Apply(input("a", 11, 1, base))
	if err != nil || len(got) != 2 || got[0].AlarmItemID != first.ID || got[1].AlarmItemID != second.ID {
		t.Fatalf("event ordering=%#v %v", got, err)
	}

	badCondition := condition("gt", 10, "warning", 0, 0)
	badCondition.ID = "33333331-3333-4333-8333-333333333333"
	bad := first
	bad.ID = "33333333-3333-4333-8333-333333333333"
	bad.DisplayName = "bad"
	bad.Conditions = []model.AlarmCondition{badCondition}
	goodState := first
	goodState.Conditions = []model.AlarmCondition{{ID: conditionID, Kind: "state", Operator: "eq", Params: json.RawMessage(`{"expected":"ok"}`), Severity: "warning", Deadband: json.RawMessage(`0`)}}
	r, err = NewRuntime(identity(), model.ProjectArtifact{AlarmItems: []model.AlarmItem{bad, goodState}}, State{})
	if err != nil {
		t.Fatal(err)
	}
	before, _ := json.Marshal(r.State())
	stringInput := PointInput{SchemaVersion: "data.raw.v1", EventID: eventID("c"), PointID: pointID, Epoch: 1, Sequence: 1, Value: json.RawMessage(`"ok"`), Quality: "good", SourceTimestamp: base, ServerTimestamp: base, ReceivedAt: base}
	if _, err = r.Apply(stringInput); err == nil {
		t.Fatal("second item type error must surface")
	}
	after, _ := json.Marshal(r.State())
	if string(before) != string(after) {
		t.Fatalf("failed Apply mutated state:\n%s\n%s", before, after)
	}
}

func TestLargeIntegersRemainDistinctForThresholdRateAndTransition(t *testing.T) {
	base := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	bigThreshold := condition("gt", 0, "warning", 0, 0)
	bigThreshold.Params = json.RawMessage(`{"threshold":9007199254740992}`)
	r := newRuntime(t, []model.AlarmCondition{bigThreshold})
	if got, err := r.Apply(rawInput("a", `9007199254740993`, 1, base)); err != nil || len(got) != 1 {
		t.Fatalf("exact threshold=%#v %v", got, err)
	}
	rate := model.AlarmCondition{ID: conditionID, Kind: "rate_of_change", Operator: "gt", Params: json.RawMessage(`{"direction":"rise","limit":0.5,"windowMs":1000}`), Severity: "warning"}
	r = newRuntime(t, []model.AlarmCondition{rate})
	r.Apply(rawInput("b", `9007199254740992`, 1, base))
	if got, err := r.Apply(rawInput("c", `9007199254740993`, 2, base.Add(time.Second))); err != nil || len(got) != 1 {
		t.Fatalf("exact rate=%#v %v", got, err)
	}
	transition := model.AlarmCondition{ID: conditionID, Kind: "transition", Operator: "changed", Params: json.RawMessage(`{}`), Severity: "warning"}
	r = newRuntime(t, []model.AlarmCondition{transition})
	r.Apply(rawInput("d", `9007199254740992`, 1, base))
	if got, err := r.Apply(rawInput("e", `9007199254740993`, 2, base.Add(time.Second))); err != nil || len(got) != 1 {
		t.Fatalf("exact transition=%#v %v", got, err)
	}
}

func TestRestoredStateIsClonedAndCorruptionFailsClosed(t *testing.T) {
	base := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	state := State{Items: map[string]ItemState{itemID: {Inputs: map[string]SampleState{pointID: {EventID: eventID("a"), Epoch: 1, Sequence: 1, Value: json.RawMessage(`1`), Quality: "good", SourceTimestamp: base, ServerTimestamp: base, ReceivedAt: base}}}}}
	r, err := NewRuntime(identity(), artifact([]model.AlarmCondition{condition("gt", 10, "warning", 0, 0)}), state)
	if err != nil {
		t.Fatal(err)
	}
	state.Items[itemID].Inputs[pointID] = SampleState{}
	if r.State().Items[itemID].Inputs[pointID].EventID == "" {
		t.Fatal("runtime retained caller state map")
	}
	corrupt := State{Items: map[string]ItemState{itemID: {ActiveConditionID: conditionID, AlarmID: "x", OpenedAt: &base, StateVersion: 1, TransitionSequence: 1, Inputs: map[string]SampleState{"unknown": {}}}}}
	if _, err := NewRuntime(identity(), artifact([]model.AlarmCondition{condition("gt", 10, "warning", 0, 0)}), corrupt); err == nil {
		t.Fatal("corrupt restored state must fail")
	}
}

func TestUpgradeDowngradeDeadbandAndSecondRaise(t *testing.T) {
	base := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	warning := condition("gt", 10, "warning", 0, 2000)
	warning.Deadband = json.RawMessage(`2`)
	critical := condition("gt", 20, "critical", 1000, 2000)
	critical.ID = "bbbbbbb1-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	critical.Deadband = json.RawMessage(`2`)
	r := newRuntime(t, []model.AlarmCondition{warning, critical})
	got, err := r.Apply(input("a", 11, 1, base))
	if err != nil || len(got) != 1 || got[0].Transition.Severity != "warning" {
		t.Fatalf("initial warning=%#v %v", got, err)
	}
	firstID := got[0].AlarmID
	if got, err = r.Apply(input("b", 21, 2, base.Add(time.Second))); err != nil || len(got) != 0 {
		t.Fatalf("upgrade delay=%#v %v", got, err)
	}
	if got, err = r.Sweep(base.Add(2 * time.Second)); err != nil || len(got) != 1 || got[0].Operation != "SEVERITY_CHANGE" || got[0].Transition.Severity != "critical" {
		t.Fatalf("upgrade=%#v %v", got, err)
	}
	// critical 的 2 单位死区使 19 仍视为匹配，不能启动降级计时。
	if got, err = r.Apply(input("c", 19, 3, base.Add(3*time.Second))); err != nil || len(got) != 0 {
		t.Fatalf("deadband=%#v %v", got, err)
	}
	if got, err = r.Apply(input("d", 15, 4, base.Add(4*time.Second))); err != nil || len(got) != 0 {
		t.Fatalf("downgrade delay=%#v %v", got, err)
	}
	if got, err = r.Sweep(base.Add(6 * time.Second)); err != nil || len(got) != 1 || got[0].Transition.Severity != "warning" {
		t.Fatalf("downgrade=%#v %v", got, err)
	}
	// warning 的死区使 9 保持活动，7 才启动清警计时。
	if got, err = r.Apply(input("e", 9, 5, base.Add(7*time.Second))); err != nil || len(got) != 0 {
		t.Fatalf("warning deadband=%#v %v", got, err)
	}
	if got, err = r.Apply(input("f", 7, 6, base.Add(8*time.Second))); err != nil || len(got) != 0 {
		t.Fatalf("clear delay=%#v %v", got, err)
	}
	if got, err = r.Sweep(base.Add(10 * time.Second)); err != nil || len(got) != 1 || got[0].Operation != "CLEAR" {
		t.Fatalf("clear=%#v %v", got, err)
	}
	if got, err = r.Apply(input("e", 11, 7, base.Add(11*time.Second))); err != nil || len(got) != 1 || got[0].Operation != "RAISE" || got[0].AlarmID == firstID || got[0].Transition.StateVersion != 5 || got[0].Transition.TransitionSequence != 5 {
		t.Fatalf("second raise identity/version=%#v %v", got, err)
	}
}

func TestPausedCandidateDoesNotAccumulateWallClock(t *testing.T) {
	base := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	r := newRuntime(t, []model.AlarmCondition{condition("gt", 10, "warning", 1000, 0)})
	if got, err := r.Apply(input("a", 11, 1, base)); err != nil || len(got) != 0 {
		t.Fatalf("candidate=%#v %v", got, err)
	}
	bad := input("b", 11, 2, base.Add(500*time.Millisecond))
	bad.Quality = "bad"
	if got, err := r.Apply(bad); err != nil || len(got) != 0 {
		t.Fatalf("pause=%#v %v", got, err)
	}
	if got, err := r.Sweep(base.Add(24 * time.Hour)); err != nil || len(got) != 0 {
		t.Fatalf("paused sweep=%#v %v", got, err)
	}
	resume := input("c", 11, 3, base.Add(24*time.Hour))
	if got, err := r.Apply(resume); err != nil || len(got) != 0 {
		t.Fatalf("resume begins new delay=%#v %v", got, err)
	}
	if got, err := r.Sweep(base.Add(24*time.Hour + 500*time.Millisecond)); err != nil || len(got) != 0 {
		t.Fatalf("resume delay shortened=%#v %v", got, err)
	}
	if got, err := r.Sweep(base.Add(24*time.Hour + time.Second)); err != nil || len(got) != 1 || got[0].Operation != "RAISE" {
		t.Fatalf("resume delayed raise=%#v %v", got, err)
	}
}

func TestConditionKindsAndRateBudgetFailClosed(t *testing.T) {
	base := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	cases := []struct {
		name      string
		condition model.AlarmCondition
		first     PointInput
		second    *PointInput
	}{
		{"gt", condition("gt", 10, "warning", 0, 0), input("a", 11, 1, base), nil},
		{"gte", condition("gte", 10, "warning", 0, 0), input("a", 10, 1, base), nil},
		{"lt", condition("lt", 10, "warning", 0, 0), input("a", 9, 1, base), nil},
		{"lte", condition("lte", 10, "warning", 0, 0), input("a", 10, 1, base), nil},
		{"range between", model.AlarmCondition{ID: conditionID, Kind: "range", Operator: "between", Params: json.RawMessage(`{"lower":10,"upper":20}`), Severity: "warning"}, input("b", 10, 1, base), nil},
		{"range outside", model.AlarmCondition{ID: conditionID, Kind: "range", Operator: "outside", Params: json.RawMessage(`{"lower":10,"upper":20}`), Severity: "warning"}, input("b", 9, 1, base), nil},
		{"state", model.AlarmCondition{ID: conditionID, Kind: "state", Operator: "eq", Params: json.RawMessage(`{"expected":"on"}`), Severity: "warning"}, rawInput("c", `"on"`, 1, base), nil},
		{"text regex", model.AlarmCondition{ID: conditionID, Kind: "text_match", Operator: "regex", Params: json.RawMessage(`{"expected":"^ALARM-[0-9]+$"}`), Severity: "warning"}, rawInput("d", `"ALARM-7"`, 1, base), nil},
		{"deviation", model.AlarmCondition{ID: conditionID, Kind: "deviation", Operator: "gt", Params: json.RawMessage(`{"baseline":10,"limit":2}`), Severity: "warning"}, input("e", 13, 1, base), nil},
		{"quality", model.AlarmCondition{ID: conditionID, Kind: "quality", Operator: "in", Params: json.RawMessage(`{"qualities":["bad"]}`), Severity: "warning"}, func() PointInput { p := input("f", 1, 1, base); p.Quality = "bad"; return p }(), nil},
		{"rise rate", model.AlarmCondition{ID: conditionID, Kind: "rate_of_change", Operator: "gt", Params: json.RawMessage(`{"direction":"rise","limit":5,"windowMs":1000}`), Severity: "warning"}, input("e", 0, 1, base), ptrInput(input("f", 10, 2, base.Add(time.Second)))},
		{"fall rate", model.AlarmCondition{ID: conditionID, Kind: "rate_of_change", Operator: "gt", Params: json.RawMessage(`{"direction":"fall","limit":5,"windowMs":1000}`), Severity: "warning"}, input("e", 10, 1, base), ptrInput(input("f", 0, 2, base.Add(time.Second)))},
		{"absolute rate", model.AlarmCondition{ID: conditionID, Kind: "rate_of_change", Operator: "gt", Params: json.RawMessage(`{"direction":"absolute","limit":5,"windowMs":1000}`), Severity: "warning"}, input("e", 10, 1, base), ptrInput(input("f", 0, 2, base.Add(time.Second)))},
		{"changed", model.AlarmCondition{ID: conditionID, Kind: "transition", Operator: "changed", Params: json.RawMessage(`{}`), Severity: "warning"}, rawInput("a", `1`, 1, base), ptrInput(rawInput("b", `2`, 2, base.Add(time.Second)))},
		{"rising", model.AlarmCondition{ID: conditionID, Kind: "transition", Operator: "rising", Params: json.RawMessage(`{}`), Severity: "warning"}, boolInput("a", false, 1, base), ptrInput(boolInput("b", true, 2, base.Add(time.Second)))},
		{"falling", model.AlarmCondition{ID: conditionID, Kind: "transition", Operator: "falling", Params: json.RawMessage(`{}`), Severity: "warning"}, boolInput("a", true, 1, base), ptrInput(boolInput("b", false, 2, base.Add(time.Second)))},
		{"from to", model.AlarmCondition{ID: conditionID, Kind: "transition", Operator: "from_to", Params: json.RawMessage(`{"from":"off","to":"on"}`), Severity: "warning"}, rawInput("a", `"off"`, 1, base), ptrInput(rawInput("b", `"on"`, 2, base.Add(time.Second)))},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newRuntime(t, []model.AlarmCondition{tc.condition})
			got, err := r.Apply(tc.first)
			if err != nil {
				t.Fatal(err)
			}
			if tc.second != nil {
				got, err = r.Apply(*tc.second)
				if err != nil {
					t.Fatal(err)
				}
			}
			if len(got) != 1 || got[0].Operation != "RAISE" || got[0].Transition.AckedAt != nil || got[0].Transition.AcknowledgedBy != nil {
				t.Fatalf("condition did not raise valid non-ACK event: %#v", got)
			}
		})
	}

	rate := model.AlarmCondition{ID: conditionID, Kind: "rate_of_change", Operator: "gt", Params: json.RawMessage(`{"direction":"absolute","limit":999999999,"windowMs":86400000}`), Severity: "warning"}
	r := newRuntime(t, []model.AlarmCondition{rate})
	for sequence := int64(1); sequence <= 4096; sequence++ {
		if _, err := r.Apply(input("a", float64(sequence), sequence, base.Add(time.Duration(sequence)*time.Nanosecond))); err != nil {
			t.Fatalf("sample %d: %v", sequence, err)
		}
	}
	if _, err := r.Apply(input("b", 4097, 4097, base.Add(4097*time.Nanosecond))); err == nil {
		t.Fatal("rate budget must fail-stop rather than silently change its window")
	}
	if got := len(r.State().Items[itemID].RateHistory); got != 4096 {
		t.Fatalf("failed rate append mutated persisted state: %d", got)
	}
}

func TestDerivedExpressionRaisesAndRejectsStringArithmetic(t *testing.T) {
	base := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	item := model.AlarmItem{ID: itemID, Revision: 1, DisplayName: "derived", Enabled: true, Mode: "derived", EvaluationMode: "single", Inputs: []model.Input{{Alias: "a", DatapointID: pointID}, {Alias: "b", DatapointID: "33333333-3333-4333-8333-333333333333"}}, DerivedExpression: ptr("a + b"), Conditions: []model.AlarmCondition{condition("gt", 10, "warning", 0, 0)}}
	r, err := NewRuntime(identity(), model.ProjectArtifact{AlarmItems: []model.AlarmItem{item}}, State{})
	if err != nil {
		t.Fatal(err)
	}
	if got, err := r.Apply(input("a", 7, 1, base)); err != nil || len(got) != 0 {
		t.Fatalf("derived first input=%#v %v", got, err)
	}
	second := input("b", 6, 1, base)
	second.PointID = "33333333-3333-4333-8333-333333333333"
	if got, err := r.Apply(second); err != nil || len(got) != 1 || got[0].Operation != "RAISE" {
		t.Fatalf("derived raise=%#v %v", got, err)
	}
	if _, err := evalExpression("a + 1", map[string]json.RawMessage{"a": json.RawMessage(`"1"`)}); err == nil {
		t.Fatal("derived string arithmetic must fail closed")
	}
}

func TestDeadbandUsesExactJSONDecimal(t *testing.T) {
	base := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	c := condition("gt", 0, "warning", 0, 0)
	c.Params = json.RawMessage(`{"threshold":9007199254740993}`)
	c.Deadband = json.RawMessage(`0.1`)
	r := newRuntime(t, []model.AlarmCondition{c})
	if got, err := r.Apply(rawInput("a", `9007199254740994`, 1, base)); err != nil || len(got) != 1 {
		t.Fatalf("large exact raise=%#v %v", got, err)
	}
	if got, err := r.Apply(rawInput("b", `9007199254740992.95`, 2, base.Add(time.Second))); err != nil || len(got) != 0 {
		t.Fatalf("0.1 deadband must retain at > threshold-deadband: %#v %v", got, err)
	}
	if got, err := r.Apply(rawInput("c", `9007199254740992.9`, 3, base.Add(2*time.Second))); err != nil || len(got) != 1 || got[0].Operation != "CLEAR" {
		t.Fatalf("exact deadband boundary=%#v %v", got, err)
	}
	c = condition("gt", 1, "warning", 0, 0)
	c.Deadband = json.RawMessage(`0.0000000000000000000001`)
	r = newRuntime(t, []model.AlarmCondition{c})
	r.Apply(rawInput("d", `1.1`, 1, base))
	if got, err := r.Apply(rawInput("e", `0.99999999999999999999995`, 2, base.Add(time.Second))); err != nil || len(got) != 0 {
		t.Fatalf("high precision deadband retained=%#v %v", got, err)
	}
	if got, err := r.Apply(rawInput("f", `0.9999999999999999999999`, 3, base.Add(2*time.Second))); err != nil || len(got) != 1 || got[0].Operation != "CLEAR" {
		t.Fatalf("high precision deadband boundary=%#v %v", got, err)
	}
	c.Deadband = json.RawMessage(`0.1 trailing`)
	if _, err := NewRuntime(identity(), artifact([]model.AlarmCondition{c}), State{}); err == nil {
		t.Fatal("trailing deadband must fail closed")
	}
}

func TestStrictConfigurationAndRestoredInvariants(t *testing.T) {
	base := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	invalid := []model.AlarmItem{
		{ID: itemID, Revision: 0, DisplayName: "bad", Enabled: true, Mode: "point", EvaluationMode: "single", Inputs: []model.Input{{Alias: "v", DatapointID: pointID}}, Conditions: []model.AlarmCondition{condition("gt", 1, "warning", 0, 0)}},
		{ID: itemID, Revision: 1, DisplayName: "bad", Enabled: true, Mode: "derived", EvaluationMode: "single", Inputs: []model.Input{{Alias: "true", DatapointID: pointID}, {Alias: "b", DatapointID: "33333333-3333-4333-8333-333333333333"}}, DerivedExpression: ptr("true + b"), Conditions: []model.AlarmCondition{condition("gt", 1, "warning", 0, 0)}},
		{ID: itemID, Revision: 1, DisplayName: "bad", Enabled: true, Mode: "derived", EvaluationMode: "single", Inputs: []model.Input{{Alias: "a", DatapointID: pointID}, {Alias: "b", DatapointID: pointID}}, DerivedExpression: ptr("a + b"), Conditions: []model.AlarmCondition{condition("gt", 1, "warning", 0, 0)}},
	}
	for _, item := range invalid {
		if _, err := NewRuntime(identity(), model.ProjectArtifact{AlarmItems: []model.AlarmItem{item}}, State{}); err == nil {
			t.Fatal("invalid configuration accepted")
		}
	}
	high := condition("gt", 20, "warning", 0, 0)
	high.ID = "bbbbbbb1-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	low := condition("lt", 10, "critical", 0, 0)
	low.ID = "ccccccc1-cccc-4ccc-8ccc-cccccccccccc"
	if _, err := NewRuntime(identity(), artifact([]model.AlarmCondition{high, low}), State{}); err == nil {
		t.Fatal("mixed highest_matching directions accepted")
	}
	corrupt := State{Items: map[string]ItemState{itemID: {CandidateConditionID: conditionID, CandidateSince: &base, ActiveConditionID: conditionID, AlarmID: eventID("a"), OpenedAt: &base, StateVersion: 1, TransitionSequence: 1, LastValue: json.RawMessage(`1`), LastQuality: "good", LastSourceAt: &base, LastEvaluatedAt: &base}}}
	if _, err := NewRuntime(identity(), artifact([]model.AlarmCondition{condition("gt", 1, "warning", 0, 0)}), corrupt); err == nil {
		t.Fatal("candidate/active overlap accepted")
	}
}

func newRuntime(t *testing.T, conditions []model.AlarmCondition) *Runtime {
	t.Helper()
	for index := range conditions {
		if len(conditions[index].Deadband) == 0 {
			conditions[index].Deadband = json.RawMessage(`0`)
		}
	}
	r, err := NewRuntime(identity(), artifact(conditions), State{})
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func identity() Identity {
	return Identity{DeploymentID: "dep", AccountID: "account", OwnerID: "alarm-owner", Epoch: 1}
}
func artifact(conditions []model.AlarmCondition) model.ProjectArtifact {
	mode := "single"
	if len(conditions) > 1 {
		mode = "highest_matching"
	}
	return model.ProjectArtifact{AlarmItems: []model.AlarmItem{{ID: itemID, Revision: 1, DisplayName: "temperature", Enabled: true, Mode: "point", EvaluationMode: mode, Inputs: []model.Input{{Alias: "v", DatapointID: pointID}}, Conditions: conditions}}}
}
func condition(op string, threshold float64, sev string, trigger, clear int64) model.AlarmCondition {
	raw, _ := json.Marshal(map[string]float64{"threshold": threshold})
	return model.AlarmCondition{ID: conditionID, Kind: "threshold", Operator: op, Params: raw, Severity: sev, TriggerDelayMS: trigger, ClearDelayMS: clear, Deadband: json.RawMessage(`0`)}
}
func input(id string, v float64, seq int64, at time.Time) PointInput {
	raw, _ := json.Marshal(v)
	return PointInput{SchemaVersion: "data.raw.v1", EventID: eventID(id), PointID: pointID, Epoch: 1, Sequence: seq, Value: raw, Quality: "good", SourceTimestamp: at, ServerTimestamp: at, ReceivedAt: at}
}
func boolInput(id string, v bool, seq int64, at time.Time) PointInput {
	raw, _ := json.Marshal(v)
	return PointInput{SchemaVersion: "data.raw.v1", EventID: eventID(id), PointID: pointID, Epoch: 1, Sequence: seq, Value: raw, Quality: "good", SourceTimestamp: at, ServerTimestamp: at, ReceivedAt: at}
}
func rawInput(id, value string, seq int64, at time.Time) PointInput {
	return PointInput{SchemaVersion: "data.raw.v1", EventID: eventID(id), PointID: pointID, Epoch: 1, Sequence: seq, Value: json.RawMessage(value), Quality: "good", SourceTimestamp: at, ServerTimestamp: at, ReceivedAt: at}
}
func eventID(seed string) string {
	out := ""
	for len(out) < 64 {
		out += seed
	}
	return out[:64]
}
func ptr(v string) *string              { return &v }
func ptrInput(v PointInput) *PointInput { return &v }
