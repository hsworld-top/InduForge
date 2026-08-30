package alarm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/indu-forge/runtime-engine/internal/eventid"
	"github.com/indu-forge/runtime-engine/internal/loader"
	"github.com/indu-forge/runtime-engine/internal/model"
)

var uuid = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
var severity = regexp.MustCompile(`^[a-z][a-z0-9_]{0,29}$`)
var stableID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
var eventIDRegex = regexp.MustCompile(`^[0-9a-f]{64}$`)

// NewRuntime 只接收已由 loader 校验的 Artifact；仍复核报警专用的参数与跨引用，避免后续接入者绕过边界。
func NewRuntime(identity Identity, artifact model.ProjectArtifact, restored State) (*Runtime, error) {
	if !stable(identity.DeploymentID) || !stable(identity.AccountID) || !stable(identity.OwnerID) || identity.Epoch < 1 {
		return nil, errors.New("alarm identity 非法")
	}
	r := &Runtime{identity: identity, items: map[string]model.AlarmItem{}, state: cloneState(restored)}
	if r.state.Items == nil {
		r.state.Items = map[string]ItemState{}
	}
	for _, item := range artifact.AlarmItems {
		if !item.Enabled {
			continue
		}
		if err := validateItem(item); err != nil {
			return nil, fmt.Errorf("alarm %s: %w", item.ID, err)
		}
		if _, exists := r.items[item.ID]; exists {
			return nil, errors.New("alarm item 重复")
		}
		r.items[item.ID] = item
		r.itemIDs = append(r.itemIDs, item.ID)
		if _, ok := r.state.Items[item.ID]; !ok {
			r.state.Items[item.ID] = ItemState{Inputs: map[string]SampleState{}}
		}
		if r.state.Items[item.ID].Inputs == nil {
			s := r.state.Items[item.ID]
			s.Inputs = map[string]SampleState{}
			r.state.Items[item.ID] = s
		}
	}
	sort.Strings(r.itemIDs)
	for id := range r.state.Items {
		if _, ok := r.items[id]; !ok {
			return nil, errors.New("restored state 包含未知 alarm")
		}
	}
	for id, item := range r.items {
		// json.Marshal encodes a nil RawMessage as JSON null. In-memory optional
		// state uses an empty slice, so normalize only that wire representation
		// before enforcing the paired last-state invariants.
		state := r.state.Items[id]
		if bytes.Equal(state.LastValue, []byte("null")) {
			state.LastValue = nil
			r.state.Items[id] = state
		}
		if err := validateRestored(item, r.state.Items[id]); err != nil {
			return nil, fmt.Errorf("alarm %s restored state: %w", id, err)
		}
	}
	return r, nil
}

func (r *Runtime) State() State { return cloneState(r.state) }

// NextEvaluationAt 返回无需新输入仍可能改变状态的最早 UTC 时刻。它只安排
// trigger/clear/stale 的确定性边界；rate 必须由新样本推进，不能靠忙轮询猜测。
func NextEvaluationAt(item model.AlarmItem, state ItemState, now time.Time) *time.Time {
	if now.IsZero() || now.Location() != time.UTC {
		return nil
	}
	var due *time.Time
	consider := func(candidate time.Time) {
		candidate = candidate.UTC()
		if candidate.Before(now) {
			candidate = now
		}
		if due == nil || candidate.Before(*due) {
			copy := candidate
			due = &copy
		}
	}
	if state.CandidateConditionID != "" && state.CandidateSince != nil {
		if c := conditionByID(item, state.CandidateConditionID); c != nil {
			consider(state.CandidateSince.Add(time.Duration(c.TriggerDelayMS) * time.Millisecond))
		}
	}
	if state.ClearSince != nil && state.ActiveConditionID != "" {
		if c := conditionByID(item, state.ActiveConditionID); c != nil {
			consider(state.ClearSince.Add(time.Duration(c.ClearDelayMS) * time.Millisecond))
		}
	}
	// stale 已经 raise 后只能由新输入恢复；继续把同一过期样本排为 due 会形成忙轮询。
	if state.ActiveConditionID == "" && state.LastSourceAt != nil {
		for _, c := range item.Conditions {
			if c.Kind != "stale" {
				continue
			}
			p, err := params(c.Params)
			if err != nil {
				return nil
			}
			if age, ok := durationParamExact(p, "maxAgeMs"); ok {
				consider(state.LastSourceAt.Add(age))
			}
		}
	}
	return due
}

// Apply 只接受严格归一化点位。重复或乱序事实不得倒退任何报警状态。
func (r *Runtime) Apply(input PointInput) ([]Event, error) {
	if err := validateInput(input); err != nil {
		return nil, err
	}
	// 一个输入会影响多个报警；任一报警配置/表达式错误时整个内存事务回滚，
	// 由后续 PostgreSQL 同一事务复用此 fail-closed 语义。
	candidate := *r
	candidate.state = cloneState(r.state)
	events, err := candidate.apply(input)
	if err != nil {
		return nil, err
	}
	r.state = candidate.state
	return events, nil
}

func (r *Runtime) apply(input PointInput) ([]Event, error) {
	var out []Event
	for _, id := range r.itemIDs {
		item := r.items[id]
		if !uses(item, input.PointID) {
			continue
		}
		s := r.state.Items[id]
		old, exists := s.Inputs[input.PointID]
		if exists && compareInput(input, old) <= 0 {
			continue
		}
		s.Inputs[input.PointID] = sample(input)
		r.state.Items[id] = s
		if e, ok, err := r.evaluate(id, item, input.ServerTimestamp, input.ReceivedAt); err != nil {
			return nil, err
		} else if ok {
			out = append(out, e)
		}
		// 仅新、良好且已归一化的事实进入时间窗；Sweep 绝不伪造样本。
		if err := r.remember(id, item); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (r *Runtime) remember(id string, item model.AlarmItem) error {
	s := r.state.Items[id]
	hasRate := false
	for _, condition := range item.Conditions {
		if condition.Kind == "rate_of_change" {
			hasRate = true
			break
		}
	}
	if !hasRate {
		return nil
	}
	value, quality, offline, source, err := itemValue(item, s.Inputs)
	if err != nil || quality != "good" || offline {
		return err
	}
	var decoded any
	decoder := json.NewDecoder(bytes.NewReader(value))
	decoder.UseNumber()
	if decoder.Decode(&decoded) != nil {
		return errors.New("报警 value 非法")
	}
	n, ok := ratAny(decoded)
	if !ok || source.IsZero() {
		return nil
	}
	if len(s.RateHistory) == 0 || source.After(s.RateHistory[len(s.RateHistory)-1].At) {
		s.RateHistory = append(s.RateHistory, RateSample{At: source, Value: n.RatString()})
		var maxWindow time.Duration
		for _, condition := range item.Conditions {
			if condition.Kind == "rate_of_change" {
				p, _ := params(condition.Params)
				if d := durationParam(p, "windowMs"); d > maxWindow {
					maxWindow = d
				}
			}
		}
		if maxWindow > 0 {
			cutoff := source.Add(-maxWindow)
			// 保留 cutoff 前最后一个样本；若整个历史都已过期，仅保留最新样本。
			keep := len(s.RateHistory) - 1
			for i := 1; i < len(s.RateHistory); i++ {
				if s.RateHistory[i].At.After(cutoff) {
					keep = i - 1
					break
				}
			}
			if keep > 0 {
				s.RateHistory = append([]RateSample(nil), s.RateHistory[keep:]...)
			}
			// 不能静默截断窗口内样本，否则 base 变化会让同一事实得到不同 rate 结论。
			// 达到状态预算时 fail-stop，由上层记录配置/流量异常后再人工处置。
			if len(s.RateHistory) > 4096 {
				return errors.New("rate history 超过 4096 条安全上限")
			}
		}
	}
	r.state.Items[id] = s
	return nil
}

// Sweep 处理无新消息时的 stale/offline 和 trigger/clear delay；now 必须是 UTC。
func (r *Runtime) Sweep(now time.Time) ([]Event, error) {
	if now.IsZero() || now.Location() != time.UTC {
		return nil, errors.New("alarm sweep 时间必须为 UTC")
	}
	candidate := *r
	candidate.state = cloneState(r.state)
	var out []Event
	for _, id := range candidate.itemIDs {
		item := candidate.items[id]
		if e, ok, err := candidate.evaluate(id, item, now, now); err != nil {
			return nil, err
		} else if ok {
			out = append(out, e)
		}
	}
	r.state = candidate.state
	return out, nil
}

func (r *Runtime) evaluate(id string, item model.AlarmItem, now, received time.Time) (Event, bool, error) {
	s := r.state.Items[id]
	previousState := s
	if len(s.Inputs) != len(item.Inputs) {
		return Event{}, false, nil
	}
	value, quality, offline, source, err := itemValue(item, s.Inputs)
	if err != nil {
		return Event{}, false, err
	}
	// A newer producer sequence may carry an earlier server timestamp.  Never
	// discard that current-master fact; evaluate against a monotonic logical
	// clock while retaining its original source/received timestamps in output.
	evaluationNow := now.UTC()
	if s.LastEvaluatedAt != nil && evaluationNow.Before(s.LastEvaluatedAt.UTC()) {
		evaluationNow = s.LastEvaluatedAt.UTC()
	}
	matched, condition, paused, err := selectCondition(item, s, value, quality, offline, source, evaluationNow)
	if err != nil {
		return Event{}, false, err
	}
	n := evaluationNow
	s.LastEvaluatedAt = &n
	s.LastValue = cloneRaw(value)
	s.LastQuality = quality
	s.LastSourceAt = &source
	var emit *Event
	if s.ActiveConditionID == "" {
		if paused {
			// 暂停不累计墙钟时间，恢复后从新的连续事实开始计算 trigger delay。
			s.CandidateConditionID, s.CandidateSince = "", nil
			// derived 的任一输入 bad/unknown 时 itemValue 没有 value。不能把
			// nil value 与本次质量/时间一起持久化，否则恢复时会违反 last state
			// 的原子不变量；输入快照仍保留，下一条 good 输入可重新求值。
			s.LastValue, s.LastQuality, s.LastSourceAt, s.LastEvaluatedAt = previousState.LastValue, previousState.LastQuality, previousState.LastSourceAt, previousState.LastEvaluatedAt
			r.state.Items[id] = s
			return Event{}, false, nil
		}
		if matched {
			if s.CandidateConditionID != condition.ID {
				s.CandidateConditionID = condition.ID
				s.CandidateSince = &n
			}
			if elapsed(s.CandidateSince, n) >= condition.TriggerDelayMS {
				emit, err = r.raise(item, &s, condition, value, quality, source, n, received)
			}
		} else {
			s.CandidateConditionID = ""
			s.CandidateSince = nil
		}
	} else {
		active := conditionByID(item, s.ActiveConditionID)
		if active == nil {
			return Event{}, false, errors.New("持久化 active condition 不存在")
		}
		activeMatched, activePaused, e := matches(*active, s, value, quality, offline, source, evaluationNow)
		if e != nil {
			return Event{}, false, e
		}
		if activePaused {
			// data_service trial 语义：bad/offline 不是恢复事实，不能触发清警或等级切换。
			s.CandidateConditionID, s.CandidateSince, s.ClearSince = "", nil, nil
			s.LastValue, s.LastQuality, s.LastSourceAt, s.LastEvaluatedAt = previousState.LastValue, previousState.LastQuality, previousState.LastSourceAt, previousState.LastEvaluatedAt
			r.state.Items[id] = s
			return Event{}, false, nil
		}
		if activeMatched {
			s.ClearSince = nil
			if matched && condition.ID != active.ID && deeper(*active, *condition) {
				if s.CandidateConditionID != condition.ID {
					s.CandidateConditionID = condition.ID
					s.CandidateSince = &n
				}
				if elapsed(s.CandidateSince, n) >= condition.TriggerDelayMS {
					emit, err = r.severityChange(item, &s, condition, value, quality, source, n, received)
				}
			} else {
				s.CandidateConditionID = ""
				s.CandidateSince = nil
			}
		} else {
			if s.ClearSince == nil {
				s.ClearSince = &n
			}
			if elapsed(s.ClearSince, n) >= active.ClearDelayMS {
				if matched {
					emit, err = r.severityChange(item, &s, condition, value, quality, source, n, received)
				} else {
					emit, err = r.clear(item, &s, *active, value, quality, source, n, received)
				}
			}
		}
	}
	if err != nil {
		return Event{}, false, err
	}
	r.state.Items[id] = s
	if emit == nil {
		return Event{}, false, nil
	}
	return *emit, true, nil
}

func (r *Runtime) raise(item model.AlarmItem, s *ItemState, c *model.AlarmCondition, value json.RawMessage, quality string, source, now, received time.Time) (*Event, error) {
	seq := s.TransitionSequence + 1
	alarmID, err := eventid.HashFields("runtime.alarm.instance.v1", r.identity.DeploymentID, item.ID, r.identity.OwnerID, fmt.Sprint(r.identity.Epoch), format(source), fmt.Sprint(seq))
	if err != nil {
		return nil, err
	}
	s.AlarmID = alarmID
	s.OpenedAt = &source
	s.ActiveConditionID = c.ID
	s.CandidateConditionID = ""
	s.CandidateSince = nil
	s.ClearSince = nil
	s.StateVersion++
	s.TransitionSequence = seq
	return r.event("RAISE", item, s, c, value, quality, source, now, received, nil)
}
func (r *Runtime) severityChange(item model.AlarmItem, s *ItemState, c *model.AlarmCondition, value json.RawMessage, quality string, source, now, received time.Time) (*Event, error) {
	old := conditionByID(item, s.ActiveConditionID)
	if old == nil {
		return nil, errors.New("active condition 不存在")
	}
	previous := old.Severity
	s.ActiveConditionID = c.ID
	s.CandidateConditionID = ""
	s.CandidateSince = nil
	s.ClearSince = nil
	s.StateVersion++
	s.TransitionSequence++
	return r.event("SEVERITY_CHANGE", item, s, c, value, quality, source, now, received, &previous)
}
func (r *Runtime) clear(item model.AlarmItem, s *ItemState, c model.AlarmCondition, value json.RawMessage, quality string, source, now, received time.Time) (*Event, error) {
	s.StateVersion++
	s.TransitionSequence++
	e, err := r.event("CLEAR", item, s, &c, value, quality, source, now, received, nil)
	if err != nil {
		return nil, err
	}
	cleared := format(now)
	e.Transition.ClearedAt = &cleared
	s.ActiveConditionID = ""
	s.CandidateConditionID = ""
	s.CandidateSince = nil
	s.ClearSince = nil
	s.AlarmID = ""
	s.OpenedAt = nil
	return e, nil
}
func (r *Runtime) event(op string, item model.AlarmItem, s *ItemState, c *model.AlarmCondition, value json.RawMessage, quality string, source, now, received time.Time, previous *string) (*Event, error) {
	points := pointIDs(item)
	id, err := eventid.AlarmTransition("alarm.event.v1", "alarm-transition", r.identity.DeploymentID, s.AlarmID, op, r.identity.OwnerID, r.identity.Epoch, format(source), item.ID, points, s.StateVersion, s.TransitionSequence)
	if err != nil {
		return nil, err
	}
	opened := ""
	if s.OpenedAt != nil {
		opened = format(*s.OpenedAt)
	}
	state := "OPEN"
	var cleared *string
	if op == "CLEAR" {
		state = "CLEARED"
		v := format(now)
		cleared = &v
	}
	e := &Event{SchemaVersion: "alarm.event.v1", Subject: "alarm.event", EventID: id, DeploymentID: r.identity.DeploymentID, AccountID: r.identity.AccountID, Kind: "alarm-transition", Operation: op, AlarmID: s.AlarmID, AlarmItemID: item.ID, PointIDs: points, OwnerID: r.identity.OwnerID, Epoch: r.identity.Epoch, SourceTimestamp: format(source), ServerTimestamp: format(now), ReceivedAt: format(received), Transition: &Transition{AlarmState: state, Severity: c.Severity, PreviousSeverity: previous, ConditionID: c.ID, Value: cloneRaw(value), Quality: quality, Message: item.DisplayName, OpenedAt: opened, UpdatedAt: format(now), ClearedAt: cleared, AckedAt: nil, AcknowledgedBy: nil, StateVersion: s.StateVersion, TransitionSequence: s.TransitionSequence}}
	if err := validateEvent(*e); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(e)
	if err != nil || loader.ValidateAlarmEvent(raw) != nil {
		return nil, errors.New("alarm event 未通过冻结 Schema")
	}
	return e, nil
}

func itemValue(item model.AlarmItem, inputs map[string]SampleState) (json.RawMessage, string, bool, time.Time, error) {
	values := map[string]json.RawMessage{}
	quality := "good"
	offline := false
	var newest time.Time
	for _, in := range item.Inputs {
		s, ok := inputs[in.DatapointID]
		if !ok {
			return nil, "", false, time.Time{}, nil
		}
		values[in.Alias] = s.Value
		if s.Quality != "good" {
			quality = s.Quality
		}
		offline = offline || s.Offline
		if s.SourceTimestamp.After(newest) {
			newest = s.SourceTimestamp
		}
	}
	if item.Mode == "derived" {
		if quality != "good" {
			return nil, quality, offline, newest, nil
		}
		v, e := evalExpression(*item.DerivedExpression, values)
		return v, quality, offline, newest, e
	}
	s := inputs[item.Inputs[0].DatapointID]
	return cloneRaw(s.Value), s.Quality, s.Offline, s.SourceTimestamp, nil
}

func selectCondition(item model.AlarmItem, s ItemState, value json.RawMessage, quality string, offline bool, source, now time.Time) (bool, *model.AlarmCondition, bool, error) {
	var selected *model.AlarmCondition
	evaluated := false
	for i := range item.Conditions {
		c := &item.Conditions[i]
		ok, paused, err := matches(*c, s, value, quality, offline, source, now)
		if err != nil {
			return false, nil, false, err
		}
		if paused {
			continue
		}
		evaluated = true
		if ok && (selected == nil || (item.EvaluationMode == "highest_matching" && deeper(*selected, *c))) {
			selected = c
		}
	}
	return selected != nil, selected, !evaluated, nil
}

func matches(c model.AlarmCondition, s ItemState, value json.RawMessage, quality string, offline bool, source, now time.Time) (bool, bool, error) {
	params, err := params(c.Params)
	if err != nil {
		return false, false, err
	}
	active := s.ActiveConditionID == c.ID
	if c.Kind == "offline" {
		return offline, false, nil
	}
	if c.Kind == "stale" {
		age := durationParam(params, "maxAgeMs")
		return !source.Add(age).After(now), false, nil
	}
	if c.Kind == "quality" {
		for _, q := range stringList(params["qualities"]) {
			if quality == q {
				return true, false, nil
			}
		}
		return false, false, nil
	}
	if offline || quality != "good" {
		return false, true, nil
	}
	if c.Kind == "rate_of_change" {
		return rateMatch(c, s, value, source, active), false, nil
	}
	if c.Kind == "transition" {
		if len(s.LastValue) == 0 {
			return false, false, nil
		}
		return transition(c, s.LastValue, value, params), false, nil
	}
	var v any
	d := json.NewDecoder(bytes.NewReader(value))
	d.UseNumber()
	if d.Decode(&v) != nil || !finiteAny(v) {
		return false, false, errors.New("报警 value 非法")
	}
	switch c.Kind {
	case "threshold":
		n, ok := ratAny(v)
		t, tok := ratAny(params["threshold"])
		if !ok || !tok {
			return false, false, errors.New("threshold 参数或值非法")
		}
		switch c.Operator {
		case "gt":
			if active {
				deadband, _ := deadbandRat(c.Deadband)
				return n.Cmp(new(big.Rat).Sub(t, deadband)) > 0, false, nil
			}
			return n.Cmp(t) > 0, false, nil
		case "gte":
			if active {
				deadband, _ := deadbandRat(c.Deadband)
				return n.Cmp(new(big.Rat).Sub(t, deadband)) >= 0, false, nil
			}
			return n.Cmp(t) >= 0, false, nil
		case "lt":
			if active {
				deadband, _ := deadbandRat(c.Deadband)
				return n.Cmp(new(big.Rat).Add(t, deadband)) < 0, false, nil
			}
			return n.Cmp(t) < 0, false, nil
		case "lte":
			if active {
				deadband, _ := deadbandRat(c.Deadband)
				return n.Cmp(new(big.Rat).Add(t, deadband)) <= 0, false, nil
			}
			return n.Cmp(t) <= 0, false, nil
		}
	case "range":
		n, ok := ratAny(v)
		lo, lok := ratAny(params["lower"])
		hi, hok := ratAny(params["upper"])
		if !ok || !lok || !hok || lo.Cmp(hi) >= 0 {
			return false, false, errors.New("range 参数或值非法")
		}
		inside := n.Cmp(lo) >= 0 && n.Cmp(hi) <= 0
		if active && c.Operator == "outside" {
			deadband, _ := deadbandRat(c.Deadband)
			return n.Cmp(new(big.Rat).Add(lo, deadband)) < 0 || n.Cmp(new(big.Rat).Sub(hi, deadband)) > 0, false, nil
		}
		if active && c.Operator == "between" {
			deadband, _ := deadbandRat(c.Deadband)
			return n.Cmp(new(big.Rat).Sub(lo, deadband)) >= 0 && n.Cmp(new(big.Rat).Add(hi, deadband)) <= 0, false, nil
		}
		if c.Operator == "between" {
			return inside, false, nil
		}
		return !inside, false, nil
	case "state":
		eq := equalValue(v, params["expected"])
		if c.Operator == "ne" {
			eq = !eq
		}
		return eq, false, nil
	case "text_match":
		x, ok := v.(string)
		expected, _ := params["expected"].(string)
		if !ok || expected == "" {
			return false, false, errors.New("text 参数或值非法")
		}
		switch c.Operator {
		case "eq":
			return x == expected, false, nil
		case "ne":
			return x != expected, false, nil
		case "contains":
			return strings.Contains(x, expected), false, nil
		case "regex":
			re, e := regexp.Compile(expected)
			if e != nil {
				return false, false, e
			}
			return re.MatchString(x), false, nil
		}
	case "deviation":
		n, ok := ratAny(v)
		base, bok := ratAny(params["baseline"])
		limit, lok := ratAny(params["limit"])
		if !ok || !bok || !lok {
			return false, false, errors.New("deviation 参数或值非法")
		}
		if active {
			deadband, _ := deadbandRat(c.Deadband)
			limit.Sub(limit, deadband)
			if limit.Sign() < 0 {
				limit.SetInt64(0)
			}
		}
		return new(big.Rat).Abs(new(big.Rat).Sub(n, base)).Cmp(limit) > 0, false, nil
	}
	return false, false, errors.New("未知报警条件")
}

func rateMatch(c model.AlarmCondition, s ItemState, value json.RawMessage, source time.Time, active bool) bool {
	current, ok := ratRaw(value)
	if !ok {
		return false
	}
	p, _ := params(c.Params)
	window, windowOK := durationParamExact(p, "windowMs")
	limit, lok := ratAny(p["limit"])
	if !lok || !windowOK || window <= 0 {
		return false
	}
	var base *RateSample
	cut := source.Add(-window)
	for i := range s.RateHistory {
		candidate := &s.RateHistory[i]
		if !candidate.At.After(cut) {
			base = candidate
		} else if base == nil {
			base = candidate
			break
		}
	}
	if base == nil || !source.After(base.At) {
		return false
	}
	baseValue, ok := new(big.Rat).SetString(base.Value)
	if !ok {
		return false
	}
	delta := new(big.Rat).Sub(current, baseValue)
	seconds := new(big.Rat).SetFrac(big.NewInt(source.Sub(base.At).Nanoseconds()), big.NewInt(int64(time.Second)))
	if seconds.Sign() <= 0 {
		return false
	}
	rate := new(big.Rat).Quo(delta, seconds)
	switch p["direction"] {
	case "fall":
		rate.Neg(rate)
	case "absolute":
		rate.Abs(rate)
	}
	if active {
		deadband, _ := deadbandRat(c.Deadband)
		limit.Sub(limit, deadband)
		if limit.Sign() < 0 {
			limit.SetInt64(0)
		}
	}
	return rate.Cmp(limit) > 0
}
func transition(c model.AlarmCondition, previous, current json.RawMessage, p map[string]any) bool {
	if len(previous) == 0 {
		return false
	}
	a, aok := decodeRawAny(previous)
	b, bok := decodeRawAny(current)
	if !aok || !bok {
		return false
	}
	switch c.Operator {
	case "changed":
		return !equalValue(a, b)
	case "rising":
		x, xok := a.(bool)
		y, yok := b.(bool)
		return xok && yok && !x && y
	case "falling":
		x, xok := a.(bool)
		y, yok := b.(bool)
		return xok && yok && x && !y
	case "from_to":
		return equalValue(a, p["from"]) && equalValue(b, p["to"])
	}
	return false
}
func decodeRawAny(raw json.RawMessage) (any, bool) {
	var value any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if d.Decode(&value) != nil {
		return nil, false
	}
	var trailing any
	return value, d.Decode(&trailing) == io.EOF
}
func uses(item model.AlarmItem, point string) bool {
	for _, in := range item.Inputs {
		if in.DatapointID == point {
			return true
		}
	}
	return false
}
func pointIDs(item model.AlarmItem) []string {
	out := make([]string, 0, len(item.Inputs))
	for _, in := range item.Inputs {
		out = append(out, in.DatapointID)
	}
	sort.Strings(out)
	return out
}
func conditionByID(item model.AlarmItem, id string) *model.AlarmCondition {
	for i := range item.Conditions {
		if item.Conditions[i].ID == id {
			return &item.Conditions[i]
		}
	}
	return nil
}
func deeper(current, candidate model.AlarmCondition) bool {
	if current.ID == candidate.ID {
		return false
	}
	if current.Kind == "threshold" && candidate.Kind == "threshold" {
		p, _ := params(current.Params)
		q, _ := params(candidate.Params)
		a, _ := ratAny(p["threshold"])
		b, _ := ratAny(q["threshold"])
		currentHigh := current.Operator == "gt" || current.Operator == "gte"
		candidateHigh := candidate.Operator == "gt" || candidate.Operator == "gte"
		currentLow := current.Operator == "lt" || current.Operator == "lte"
		candidateLow := candidate.Operator == "lt" || candidate.Operator == "lte"
		if a == nil || b == nil {
			return false
		}
		if currentHigh && candidateHigh {
			return b.Cmp(a) > 0
		}
		if currentLow && candidateLow {
			return b.Cmp(a) < 0
		}
	}
	return false
}
func sample(i PointInput) SampleState {
	return SampleState{EventID: i.EventID, Epoch: i.Epoch, Sequence: i.Sequence, Value: cloneRaw(i.Value), Quality: i.Quality, Offline: i.Offline, SourceTimestamp: i.SourceTimestamp, ServerTimestamp: i.ServerTimestamp, ReceivedAt: i.ReceivedAt}
}
func compareInput(in PointInput, old SampleState) int {
	if in.Epoch != old.Epoch {
		if in.Epoch > old.Epoch {
			return 1
		}
		return -1
	}
	if !in.SourceTimestamp.Equal(old.SourceTimestamp) {
		if in.SourceTimestamp.After(old.SourceTimestamp) {
			return 1
		}
		return -1
	}
	if in.Sequence != old.Sequence {
		if in.Sequence > old.Sequence {
			return 1
		}
		return -1
	}
	return strings.Compare(in.EventID, old.EventID)
}
func elapsed(from *time.Time, to time.Time) int64 {
	if from == nil {
		return 0
	}
	return to.Sub(*from).Milliseconds()
}
func cloneRaw(v json.RawMessage) json.RawMessage { return append(json.RawMessage(nil), v...) }
func format(t time.Time) string                  { return t.UTC().Format(time.RFC3339Nano) }
func durationParam(p map[string]any, key string) time.Duration {
	d, ok := durationParamExact(p, key)
	if !ok {
		return 0
	}
	return d
}
func durationParamExact(p map[string]any, key string) (time.Duration, bool) {
	r, ok := ratAny(p[key])
	if !ok || r.Sign() <= 0 || r.Denom().Cmp(big.NewInt(1)) != 0 {
		return 0, false
	}
	max := new(big.Int).Quo(big.NewInt(int64(^uint64(0)>>1)), big.NewInt(int64(time.Millisecond)))
	if r.Num().Cmp(max) > 0 {
		return 0, false
	}
	return time.Duration(r.Num().Int64()) * time.Millisecond, true
}
func ratRaw(raw json.RawMessage) (*big.Rat, bool) {
	var value any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if d.Decode(&value) != nil {
		return nil, false
	}
	var trailing any
	if d.Decode(&trailing) != io.EOF {
		return nil, false
	}
	return ratAny(value)
}
func ratAny(value any) (*big.Rat, bool) {
	var text string
	switch v := value.(type) {
	case *big.Rat:
		return new(big.Rat).Set(v), true
	case json.Number:
		text = v.String()
	case float64:
		text = strconv.FormatFloat(v, 'g', -1, 64)
	case int64:
		text = strconv.FormatInt(v, 10)
	case int:
		text = strconv.Itoa(v)
	default:
		return nil, false
	}
	r, ok := new(big.Rat).SetString(text)
	return r, ok
}
func deadbandRat(raw json.RawMessage) (*big.Rat, bool) {
	r, ok := ratRaw(raw)
	return r, ok && r.Sign() >= 0
}
func params(raw json.RawMessage) (map[string]any, error) {
	var p map[string]any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if d.Decode(&p) != nil || p == nil || !finiteAny(p) {
		return nil, errors.New("alarm params 非法")
	}
	var trailing any
	if d.Decode(&trailing) != io.EOF {
		return nil, errors.New("alarm params 含尾随 JSON")
	}
	return p, nil
}
func stringList(v any) []string {
	a, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(a))
	for _, x := range a {
		s, ok := x.(string)
		if !ok {
			return nil
		}
		out = append(out, s)
	}
	return out
}
func stable(v string) bool { return stableID.MatchString(v) }
func validateInput(i PointInput) error {
	if (i.SchemaVersion != "data.raw.v1" && i.SchemaVersion != "data.computed.v1") || !uuid.MatchString(i.PointID) || !eventIDRegex.MatchString(i.EventID) || i.Epoch < 1 || i.Sequence < 0 || !finiteRaw(i.Value) || (i.Quality != "good" && i.Quality != "bad" && i.Quality != "unknown") || i.SourceTimestamp.IsZero() || i.ServerTimestamp.IsZero() || i.ReceivedAt.IsZero() || i.SourceTimestamp.Location() != time.UTC || i.ServerTimestamp.Location() != time.UTC || i.ReceivedAt.Location() != time.UTC {
		return errors.New("alarm point input 非法")
	}
	return nil
}
func validateItem(item model.AlarmItem) error {
	if !uuid.MatchString(item.ID) || len(item.DisplayName) == 0 || len([]rune(item.DisplayName)) > 2048 || (item.Mode != "point" && item.Mode != "derived") || (item.EvaluationMode != "single" && item.EvaluationMode != "highest_matching") || len(item.Inputs) == 0 || (item.Mode == "point" && len(item.Inputs) != 1) || (item.Mode == "derived" && (len(item.Inputs) < 2 || item.DerivedExpression == nil || item.EvaluationMode != "single")) || len(item.Conditions) == 0 {
		return errors.New("alarm item 配置非法")
	}
	seenAlias := map[string]bool{}
	seenPoint := map[string]bool{}
	for _, in := range item.Inputs {
		if !uuid.MatchString(in.DatapointID) || in.Alias == "" || in.Alias == "true" || in.Alias == "false" || seenAlias[in.Alias] || seenPoint[in.DatapointID] {
			return errors.New("alarm input 配置非法")
		}
		seenAlias[in.Alias] = true
		seenPoint[in.DatapointID] = true
	}
	if item.Mode == "derived" {
		if err := validateExpression(*item.DerivedExpression, seenAlias); err != nil {
			return err
		}
	}
	conditionIDs := map[string]bool{}
	for _, c := range item.Conditions {
		if item.Revision < 1 || !uuid.MatchString(c.ID) || !severity.MatchString(c.Severity) || c.TriggerDelayMS < 0 || c.ClearDelayMS < 0 {
			return errors.New("alarm condition 配置非法")
		}
		if _, ok := deadbandRat(c.Deadband); !ok {
			return errors.New("alarm deadband 必须为有限非负精确数字")
		}
		if conditionIDs[c.ID] {
			return errors.New("alarm condition 重复")
		}
		conditionIDs[c.ID] = true
		if item.EvaluationMode == "highest_matching" && c.Kind != "threshold" {
			return errors.New("highest_matching 仅允许 threshold")
		}
		if item.Mode == "derived" && (c.Kind == "offline" || c.Kind == "quality" || c.Kind == "stale") {
			return errors.New("derived alarm 禁止设备状态条件")
		}
		if err := validateCondition(c); err != nil {
			return err
		}
	}
	if item.EvaluationMode == "highest_matching" {
		var direction string
		var previous *big.Rat
		for _, c := range item.Conditions {
			currentDirection := "high"
			if c.Operator == "lt" || c.Operator == "lte" {
				currentDirection = "low"
			}
			if direction == "" {
				direction = currentDirection
			} else if direction != currentDirection {
				return errors.New("highest_matching 不允许混合高低限")
			}
			p, _ := params(c.Params)
			threshold, _ := ratAny(p["threshold"])
			if previous != nil && ((direction == "high" && threshold.Cmp(previous) <= 0) || (direction == "low" && threshold.Cmp(previous) >= 0)) {
				return errors.New("highest_matching 阈值必须按越限方向严格递进")
			}
			previous = threshold
		}
	}
	return nil
}

// validateEvent 是输出端的最小 Schema 镜像；它在构造点 fail-closed，避免业务层把不完整事件交给 outbox。
func validateEvent(e Event) error {
	if e.SchemaVersion != "alarm.event.v1" || e.Subject != "alarm.event" || !eventIDRegex.MatchString(e.EventID) || !stable(e.DeploymentID) || !stable(e.AccountID) || !stable(e.OwnerID) || e.Epoch < 1 || e.Kind != "alarm-transition" || (e.Operation != "RAISE" && e.Operation != "SEVERITY_CHANGE" && e.Operation != "CLEAR") || !eventIDRegex.MatchString(e.AlarmID) || !uuid.MatchString(e.AlarmItemID) || len(e.PointIDs) == 0 || e.Transition == nil {
		return errors.New("alarm event 基础字段非法")
	}
	for i, point := range e.PointIDs {
		if !uuid.MatchString(point) || (i > 0 && e.PointIDs[i-1] >= point) {
			return errors.New("alarm event pointIds 非法")
		}
	}
	t := e.Transition
	if !uuid.MatchString(t.ConditionID) || !severity.MatchString(t.Severity) || len([]rune(t.Message)) == 0 || len([]rune(t.Message)) > 2048 || !finiteRaw(t.Value) || (t.Quality != "good" && t.Quality != "bad" && t.Quality != "unknown") || t.StateVersion < 1 || t.TransitionSequence < 1 {
		return errors.New("alarm event transition 非法")
	}
	parseUTC := func(v string) bool {
		x, err := time.Parse(time.RFC3339Nano, v)
		return err == nil && x.Location() == time.UTC && strings.HasSuffix(v, "Z")
	}
	if !parseUTC(e.SourceTimestamp) || !parseUTC(e.ServerTimestamp) || !parseUTC(e.ReceivedAt) || !parseUTC(t.OpenedAt) || !parseUTC(t.UpdatedAt) {
		return errors.New("alarm event 时间非法")
	}
	switch e.Operation {
	case "RAISE":
		if t.AlarmState != "OPEN" || t.PreviousSeverity != nil || t.ClearedAt != nil || t.AckedAt != nil || t.AcknowledgedBy != nil {
			return errors.New("RAISE 状态机非法")
		}
	case "SEVERITY_CHANGE":
		if t.AlarmState != "OPEN" || t.PreviousSeverity == nil || !severity.MatchString(*t.PreviousSeverity) || t.ClearedAt != nil || t.AckedAt != nil || t.AcknowledgedBy != nil {
			return errors.New("SEVERITY_CHANGE 状态机非法")
		}
	case "CLEAR":
		if t.AlarmState != "CLEARED" || t.PreviousSeverity != nil || t.ClearedAt == nil || !parseUTC(*t.ClearedAt) || t.AckedAt != nil || t.AcknowledgedBy != nil {
			return errors.New("CLEAR 状态机非法")
		}
	}
	return nil
}

func validateCondition(c model.AlarmCondition) error {
	p, e := params(c.Params)
	if e != nil {
		return e
	}
	exact := func(keys ...string) bool {
		if len(p) != len(keys) {
			return false
		}
		for _, key := range keys {
			if _, ok := p[key]; !ok {
				return false
			}
		}
		return true
	}
	numeric := func(key string) bool { _, ok := ratAny(p[key]); return ok }
	switch c.Kind {
	case "threshold":
		if !(c.Operator == "gt" || c.Operator == "gte" || c.Operator == "lt" || c.Operator == "lte") || !exact("threshold") || !numeric("threshold") {
			return errors.New("threshold 条件非法")
		}
	case "range":
		lo, lok := ratAny(p["lower"])
		hi, hok := ratAny(p["upper"])
		if !(c.Operator == "between" || c.Operator == "outside") || !exact("lower", "upper") || !lok || !hok || lo.Cmp(hi) >= 0 {
			return errors.New("range 条件非法")
		}
	case "state":
		if !(c.Operator == "eq" || c.Operator == "ne") || !exact("expected") {
			return errors.New("state 条件非法")
		}
	case "transition":
		if !(c.Operator == "changed" || c.Operator == "rising" || c.Operator == "falling" || c.Operator == "from_to") || c.TriggerDelayMS != 0 {
			return errors.New("transition 条件非法")
		}
		if c.Operator == "from_to" {
			if !exact("from", "to") || equalValue(p["from"], p["to"]) {
				return errors.New("transition from_to 条件非法")
			}
		} else if !exact() {
			return errors.New("transition params 必须为空")
		}
	case "text_match":
		expected, ok := p["expected"].(string)
		if !(c.Operator == "eq" || c.Operator == "ne" || c.Operator == "contains" || c.Operator == "regex") || !exact("expected") || !ok || strings.TrimSpace(expected) == "" {
			return errors.New("text_match 条件非法")
		}
		if c.Operator == "regex" {
			if _, err := regexp.Compile(expected); err != nil {
				return errors.New("text_match regex 非法")
			}
		}
	case "rate_of_change":
		direction, ok := p["direction"].(string)
		limit, lok := ratAny(p["limit"])
		_, wok := durationParamExact(p, "windowMs")
		if c.Operator != "gt" || !exact("direction", "limit", "windowMs") || !ok || (direction != "rise" && direction != "fall" && direction != "absolute") || !lok || limit.Sign() <= 0 || !wok {
			return errors.New("rate_of_change 条件非法")
		}
	case "deviation":
		limit, lok := ratAny(p["limit"])
		if c.Operator != "gt" || !exact("baseline", "limit") || !numeric("baseline") || !lok || limit.Sign() < 0 {
			return errors.New("deviation 条件非法")
		}
	case "quality":
		qualities := stringList(p["qualities"])
		seen := map[string]bool{}
		if c.Operator != "in" || !exact("qualities") || len(qualities) == 0 {
			return errors.New("quality 条件非法")
		}
		for _, q := range qualities {
			if (q != "bad" && q != "unknown") || seen[q] {
				return errors.New("quality 条件非法")
			}
			seen[q] = true
		}
	case "offline":
		if c.Operator != "is" || !exact() {
			return errors.New("offline 条件非法")
		}
	case "stale":
		_, ok := durationParamExact(p, "maxAgeMs")
		if c.Operator != "age_gte" || !exact("maxAgeMs") || !ok {
			return errors.New("stale 条件非法")
		}
	default:
		return errors.New("未知 alarm condition")
	}
	deadband, _ := deadbandRat(c.Deadband)
	if deadband.Sign() > 0 && !(c.Kind == "threshold" || c.Kind == "range" || c.Kind == "rate_of_change" || c.Kind == "deviation") {
		return errors.New("deadband 不适用于当前条件")
	}
	// 此白名单拒绝任何未来未经契约审计的 params 字段，不能把 JSON map 当作扩展执行入口。
	return nil
}
func cloneState(in State) State {
	out := State{Items: map[string]ItemState{}}
	for id, state := range in.Items {
		copy := state
		copy.Inputs = map[string]SampleState{}
		for point, sample := range state.Inputs {
			sample.Value = cloneRaw(sample.Value)
			copy.Inputs[point] = sample
		}
		copy.LastValue = cloneRaw(state.LastValue)
		copy.RateHistory = append([]RateSample(nil), state.RateHistory...)
		copy.CandidateSince = copyTime(state.CandidateSince)
		copy.ClearSince = copyTime(state.ClearSince)
		copy.OpenedAt = copyTime(state.OpenedAt)
		copy.LastSourceAt = copyTime(state.LastSourceAt)
		copy.LastEvaluatedAt = copyTime(state.LastEvaluatedAt)
		out.Items[id] = copy
	}
	return out
}
func copyTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := value.UTC()
	return &copy
}
func validateRestored(item model.AlarmItem, s ItemState) error {
	allowed := map[string]bool{}
	for _, in := range item.Inputs {
		allowed[in.DatapointID] = true
	}
	for point, sample := range s.Inputs {
		if !allowed[point] || !eventIDRegex.MatchString(sample.EventID) || sample.Epoch < 1 || sample.Sequence < 0 || !finiteRaw(sample.Value) || (sample.Quality != "good" && sample.Quality != "bad" && sample.Quality != "unknown") || sample.SourceTimestamp.IsZero() || sample.ServerTimestamp.IsZero() || sample.ReceivedAt.IsZero() || sample.SourceTimestamp.Location() != time.UTC || sample.ServerTimestamp.Location() != time.UTC || sample.ReceivedAt.Location() != time.UTC {
			return errors.New("input state 非法")
		}
	}
	if (s.ActiveConditionID == "") != (s.AlarmID == "") {
		return errors.New("active/alarmId 不变量非法")
	}
	if s.StateVersion < 0 || s.TransitionSequence < 0 || s.StateVersion != s.TransitionSequence {
		return errors.New("state version/sequence 不变量非法")
	}
	if s.ActiveConditionID != "" {
		if conditionByID(item, s.ActiveConditionID) == nil || !eventIDRegex.MatchString(s.AlarmID) || s.OpenedAt == nil || s.OpenedAt.Location() != time.UTC || s.StateVersion < 1 || s.TransitionSequence < 1 || s.StateVersion != s.TransitionSequence {
			return errors.New("active state 非法")
		}
	} else if s.OpenedAt != nil {
		return errors.New("inactive state 不得保留 openedAt")
	}
	if (s.CandidateConditionID == "") != (s.CandidateSince == nil) {
		return errors.New("candidate state 不变量非法")
	}
	if s.CandidateConditionID != "" && (conditionByID(item, s.CandidateConditionID) == nil || s.CandidateSince.Location() != time.UTC || s.CandidateConditionID == s.ActiveConditionID) {
		return errors.New("candidate state 非法")
	}
	if s.ClearSince != nil && (s.ActiveConditionID == "" || s.ClearSince.Location() != time.UTC) {
		return errors.New("clear state 非法")
	}
	if s.ClearSince != nil && s.CandidateSince != nil {
		return errors.New("clear 与 candidate 不得同时存在")
	}
	if (len(s.LastValue) == 0) != (s.LastSourceAt == nil || s.LastEvaluatedAt == nil) || (s.LastSourceAt == nil) != (s.LastEvaluatedAt == nil) {
		return errors.New("last state 不变量非法")
	}
	if len(s.LastValue) == 0 && s.LastQuality != "" {
		return errors.New("无 last value 不得保留质量")
	}
	if len(s.LastValue) > 0 && (!finiteRaw(s.LastValue) || (s.LastQuality != "good" && s.LastQuality != "bad" && s.LastQuality != "unknown") || s.LastSourceAt.Location() != time.UTC || s.LastEvaluatedAt.Location() != time.UTC) {
		return errors.New("last value 非法")
	}
	hasRate := false
	var maxWindow time.Duration
	for _, c := range item.Conditions {
		if c.Kind != "rate_of_change" {
			continue
		}
		hasRate = true
		p, _ := params(c.Params)
		if window := durationParam(p, "windowMs"); window > maxWindow {
			maxWindow = window
		}
	}
	if !hasRate && len(s.RateHistory) != 0 {
		return errors.New("非 rate 条件不得保留 rate history")
	}
	if len(s.RateHistory) > 4096 {
		return errors.New("rate history 超过安全上限")
	}
	var last time.Time
	for index, rate := range s.RateHistory {
		if rate.At.IsZero() || rate.At.Location() != time.UTC || !last.IsZero() && !rate.At.After(last) {
			return errors.New("rate history 时间非法")
		}
		if _, ok := new(big.Rat).SetString(rate.Value); !ok {
			return errors.New("rate history 数值非法")
		}
		if index > 0 && index < len(s.RateHistory)-1 && !rate.At.After(s.RateHistory[len(s.RateHistory)-1].At.Add(-maxWindow)) {
			return errors.New("rate history 窗口非法")
		}
		last = rate.At
	}
	return nil
}
