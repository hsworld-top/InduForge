package service

import (
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	alarmEvaluationMatched    = "matched"
	alarmEvaluationNotMatched = "not_matched"
	alarmEvaluationPaused     = "paused"
)

type alarmTrialEvaluation struct {
	state  string
	reason string
	rate   *float64
}

type alarmRateSample struct {
	timestamp time.Time
	value     float64
}

type alarmTrialMachine struct {
	input            SaveAlarmItemInput
	active           *AlarmCondition
	candidate        *AlarmCondition
	candidateElapsed int64
	clearElapsed     int64
	clearing         bool
	lastObservedAt   time.Time
	previousValue    any
	hasPreviousValue bool
	rateSamples      []alarmRateSample
	missingSince     *time.Time
	paused           bool
}

func runAlarmTrial(input SaveAlarmItemInput, samples []AlarmTrialSample) (*AlarmItemTrialResult, error) {
	machine := &alarmTrialMachine{input: input}
	result := &AlarmItemTrialResult{State: "not_triggered", Steps: make([]AlarmTrialStep, 0, len(samples))}
	for index, sample := range samples {
		if sample.ObservedAt.IsZero() {
			return nil, badAlarm(fmt.Sprintf("第 %d 个试算样本缺少 observedAt", index+1))
		}
		if !machine.lastObservedAt.IsZero() && !sample.ObservedAt.After(machine.lastObservedAt) {
			return nil, badAlarm("试算样本 observedAt 必须严格递增")
		}
		if sample.SourceTimestamp != nil && sample.SourceTimestamp.After(sample.ObservedAt) {
			return nil, badAlarm("试算样本 sourceTimestamp 不能晚于 observedAt")
		}
		step, err := machine.consume(sample)
		if err != nil {
			return nil, err
		}
		result.Steps = append(result.Steps, step)
	}
	if machine.active != nil {
		copy := *machine.active
		result.Triggered = true
		result.State = "triggered"
		result.SelectedCondition = &copy
	}
	return result, nil
}

func (m *alarmTrialMachine) consume(sample AlarmTrialSample) (AlarmTrialStep, error) {
	delta := int64(0)
	if !m.lastObservedAt.IsZero() {
		delta = sample.ObservedAt.Sub(m.lastObservedAt).Milliseconds()
	}
	if m.paused {
		delta = 0
		m.paused = false
	}
	value := sample.Value
	if m.input.Mode == "derived" {
		if len(sample.Inputs) == 0 {
			return AlarmTrialStep{}, badAlarm("组合报警试算样本必须提供 inputs")
		}
		derived, err := evaluateDerivedAlarmExpression(m.input.DerivedExpression, sample.Inputs)
		if err != nil {
			return AlarmTrialStep{}, err
		}
		value = derived
	}

	evaluations := make(map[string]alarmTrialEvaluation, len(m.input.Conditions))
	var selected *AlarmCondition
	var selectedEval alarmTrialEvaluation
	var calculatedRate *float64
	pauseReason := "quality_paused"
	allPaused := true
	for index := range m.input.Conditions {
		condition := &m.input.Conditions[index]
		active := m.active != nil && m.active.ID == condition.ID
		evaluation := m.evaluate(*condition, value, sample, active)
		evaluations[condition.ID] = evaluation
		if evaluation.rate != nil {
			calculatedRate = evaluation.rate
		}
		if evaluation.state == alarmEvaluationPaused {
			pauseReason = evaluation.reason
		}
		if evaluation.state != alarmEvaluationPaused {
			allPaused = false
		}
		if evaluation.state == alarmEvaluationMatched && alarmConditionIsDeeper(selected, condition) {
			selected = condition
			selectedEval = evaluation
		}
	}

	step := AlarmTrialStep{ObservedAt: sample.ObservedAt, SourceTimestamp: sample.SourceTimestamp, Value: value, Inputs: sample.Inputs, EvaluationState: alarmEvaluationNotMatched, Reason: "evaluated", State: "normal", CalculatedRate: calculatedRate}
	if allPaused {
		step.EvaluationState = alarmEvaluationPaused
		step.Reason = pauseReason
		step.State = "paused"
		m.paused = true
		m.finishStep(&step, sample, value, false)
		return step, nil
	}
	if selected != nil {
		step.EvaluationState = alarmEvaluationMatched
		step.Reason = selectedEval.reason
		step.CalculatedRate = selectedEval.rate
	}

	if m.active == nil {
		m.clearElapsed = 0
		m.clearing = false
		if selected == nil {
			m.candidate = nil
			m.candidateElapsed = 0
		} else {
			m.advanceCandidate(selected, delta)
			if selected.TriggerDelayMS == 0 || m.candidateElapsed >= selected.TriggerDelayMS {
				m.activate(selected)
			}
		}
	} else {
		activeEval := evaluations[m.active.ID]
		if activeEval.state == alarmEvaluationPaused {
			step.EvaluationState = alarmEvaluationPaused
			step.Reason = activeEval.reason
			step.State = "paused"
			m.paused = true
			m.finishStep(&step, sample, value, false)
			return step, nil
		}
		if activeEval.state == alarmEvaluationMatched {
			m.clearElapsed = 0
			m.clearing = false
			if selected != nil && selected.ID != m.active.ID && alarmConditionIsDeeper(m.active, selected) {
				m.advanceCandidate(selected, delta)
				if selected.TriggerDelayMS == 0 || m.candidateElapsed >= selected.TriggerDelayMS {
					m.activate(selected)
				}
			} else {
				m.candidate = nil
				m.candidateElapsed = 0
			}
		} else {
			m.candidate = nil
			m.candidateElapsed = 0
			if !m.clearing {
				m.clearing = true
				m.clearElapsed = 0
			} else {
				m.clearElapsed += delta
			}
			if m.active.ClearDelayMS == 0 || m.clearElapsed >= m.active.ClearDelayMS {
				if selected != nil {
					m.activate(selected)
				} else {
					m.active = nil
					m.clearElapsed = 0
				}
			}
		}
	}

	m.finishStep(&step, sample, value, true)
	return step, nil
}

func (m *alarmTrialMachine) evaluate(condition AlarmCondition, value any, sample AlarmTrialSample, active bool) alarmTrialEvaluation {
	if condition.Kind == "quality" {
		qualities, _ := condition.Params["qualities"].([]string)
		if qualities == nil {
			if raw, ok := condition.Params["qualities"].([]any); ok {
				for _, item := range raw {
					qualities = append(qualities, fmt.Sprint(item))
				}
			}
		}
		for _, quality := range qualities {
			if quality == normalizeAlarmQuality(sample.Quality) {
				return alarmTrialEvaluation{state: alarmEvaluationMatched, reason: "evaluated"}
			}
		}
		return alarmTrialEvaluation{state: alarmEvaluationNotMatched, reason: "evaluated"}
	}
	if condition.Kind == "offline" {
		if sample.Offline {
			return alarmTrialEvaluation{state: alarmEvaluationMatched, reason: "offline"}
		}
		return alarmTrialEvaluation{state: alarmEvaluationNotMatched, reason: "evaluated"}
	}
	if condition.Kind == "stale" {
		maxAge, ok := anyFloat(condition.Params["maxAgeMs"])
		if !ok || maxAge <= 0 {
			return alarmTrialEvaluation{state: alarmEvaluationPaused, reason: "invalid_stale_age"}
		}
		if sample.SourceTimestamp != nil {
			m.missingSince = nil
			if sample.ObservedAt.Sub(*sample.SourceTimestamp).Milliseconds() >= int64(maxAge) {
				return alarmTrialEvaluation{state: alarmEvaluationMatched, reason: "stale"}
			}
			return alarmTrialEvaluation{state: alarmEvaluationNotMatched, reason: "evaluated"}
		}
		if m.missingSince == nil {
			value := sample.ObservedAt
			m.missingSince = &value
		}
		if sample.ObservedAt.Sub(*m.missingSince).Milliseconds() >= int64(maxAge) {
			return alarmTrialEvaluation{state: alarmEvaluationMatched, reason: "missing_timestamp_stale"}
		}
		return alarmTrialEvaluation{state: alarmEvaluationNotMatched, reason: "insufficient_timestamp"}
	}
	if sample.Offline {
		return alarmTrialEvaluation{state: alarmEvaluationPaused, reason: "offline_paused"}
	}
	if normalizeAlarmQuality(sample.Quality) != "good" {
		return alarmTrialEvaluation{state: alarmEvaluationPaused, reason: "quality_paused"}
	}
	if condition.Kind == "rate_of_change" {
		return m.evaluateRate(condition, value, sample, active)
	}
	context := map[string]any{"alarmActive": active}
	if m.hasPreviousValue {
		context["previousValue"] = m.previousValue
	}
	matched, reason := evaluateAlarmCondition(condition, value, "good", false, context)
	if reason == "insufficient_previous_value" {
		return alarmTrialEvaluation{state: alarmEvaluationNotMatched, reason: "insufficient_previous_sample"}
	}
	if strings.HasPrefix(reason, "invalid_") {
		return alarmTrialEvaluation{state: alarmEvaluationPaused, reason: reason}
	}
	if matched {
		return alarmTrialEvaluation{state: alarmEvaluationMatched, reason: reason}
	}
	return alarmTrialEvaluation{state: alarmEvaluationNotMatched, reason: reason}
}

func (m *alarmTrialMachine) evaluateRate(condition AlarmCondition, value any, sample AlarmTrialSample, active bool) alarmTrialEvaluation {
	current, ok := anyFloat(value)
	window, windowOK := anyFloat(condition.Params["windowMs"])
	limit, limitOK := anyFloat(condition.Params["limit"])
	if !ok || !windowOK || !limitOK || window <= 0 {
		return alarmTrialEvaluation{state: alarmEvaluationPaused, reason: "invalid_numeric_value"}
	}
	if sample.SourceTimestamp == nil {
		return alarmTrialEvaluation{state: alarmEvaluationNotMatched, reason: "insufficient_timestamp"}
	}
	cutoff := sample.SourceTimestamp.Add(-time.Duration(window) * time.Millisecond)
	var base *alarmRateSample
	for index := range m.rateSamples {
		candidate := &m.rateSamples[index]
		if !candidate.timestamp.After(cutoff) {
			base = candidate
			continue
		}
		if base == nil {
			base = candidate
		}
		break
	}
	if base == nil || !sample.SourceTimestamp.After(base.timestamp) {
		return alarmTrialEvaluation{state: alarmEvaluationNotMatched, reason: "insufficient_previous_sample"}
	}
	seconds := sample.SourceTimestamp.Sub(base.timestamp).Seconds()
	delta := current - base.value
	direction, _ := condition.Params["direction"].(string)
	metric := math.Abs(delta) / seconds
	if direction == "rise" {
		metric = delta / seconds
	}
	if direction == "fall" {
		metric = -delta / seconds
	}
	threshold := limit
	if active {
		threshold = math.Max(limit-condition.Deadband, 0)
	}
	matched := metric > threshold
	return alarmTrialEvaluation{state: map[bool]string{true: alarmEvaluationMatched, false: alarmEvaluationNotMatched}[matched], reason: "evaluated", rate: &metric}
}

func (m *alarmTrialMachine) advanceCandidate(condition *AlarmCondition, delta int64) {
	if m.candidate == nil || m.candidate.ID != condition.ID {
		copy := *condition
		m.candidate = &copy
		m.candidateElapsed = 0
		return
	}
	m.candidateElapsed += delta
}

func (m *alarmTrialMachine) activate(condition *AlarmCondition) {
	copy := *condition
	m.active = &copy
	m.candidate = nil
	m.candidateElapsed = 0
	m.clearElapsed = 0
	m.clearing = false
}

func (m *alarmTrialMachine) finishStep(step *AlarmTrialStep, sample AlarmTrialSample, value any, updateHistory bool) {
	if m.active != nil {
		copy := *m.active
		step.ActiveCondition = &copy
		step.State = "triggered"
		if m.clearing {
			step.State = "pending_clear"
		}
	}
	if m.candidate != nil {
		copy := *m.candidate
		step.CandidateCondition = &copy
		step.CandidateElapsedMS = m.candidateElapsed
		step.RemainingTriggerDelayMS = max(m.candidate.TriggerDelayMS-m.candidateElapsed, 0)
		if m.active == nil {
			step.State = "pending_trigger"
		}
	}
	step.ClearElapsedMS = m.clearElapsed
	if m.active != nil {
		step.RemainingClearDelayMS = max(m.active.ClearDelayMS-m.clearElapsed, 0)
	}
	m.lastObservedAt = sample.ObservedAt
	if updateHistory && !sample.Offline && normalizeAlarmQuality(sample.Quality) == "good" {
		m.previousValue = value
		m.hasPreviousValue = true
		if sample.SourceTimestamp != nil {
			if numeric, ok := anyFloat(value); ok && (len(m.rateSamples) == 0 || sample.SourceTimestamp.After(m.rateSamples[len(m.rateSamples)-1].timestamp)) {
				m.rateSamples = append(m.rateSamples, alarmRateSample{timestamp: *sample.SourceTimestamp, value: numeric})
				m.trimRateSamples()
			}
		}
	}
}

func (m *alarmTrialMachine) trimRateSamples() {
	if len(m.rateSamples) < 2 {
		return
	}
	maxWindowMS := float64(0)
	for _, condition := range m.input.Conditions {
		if condition.Kind != "rate_of_change" {
			continue
		}
		if windowMS, ok := anyFloat(condition.Params["windowMs"]); ok && windowMS > maxWindowMS {
			maxWindowMS = windowMS
		}
	}
	if maxWindowMS <= 0 {
		m.rateSamples = nil
		return
	}
	cutoff := m.rateSamples[len(m.rateSamples)-1].timestamp.Add(-time.Duration(maxWindowMS) * time.Millisecond)
	// 保留窗口边界前最后一个样本，确保窗口内没有精确边界样本时仍可计算首尾变化率。
	keepFrom := 0
	for index := 1; index < len(m.rateSamples); index++ {
		if m.rateSamples[index].timestamp.After(cutoff) {
			keepFrom = index - 1
			break
		}
	}
	if keepFrom > 0 {
		m.rateSamples = append([]alarmRateSample(nil), m.rateSamples[keepFrom:]...)
	}
}

func alarmConditionIsDeeper(current, candidate *AlarmCondition) bool {
	if candidate == nil {
		return false
	}
	if current == nil {
		return true
	}
	if current.ID == candidate.ID {
		return false
	}
	currentDirection, currentThreshold := thresholdDirectionAndValue(*current)
	candidateDirection, candidateThreshold := thresholdDirectionAndValue(*candidate)
	if currentDirection == "high" && candidateDirection == "high" {
		return candidateThreshold > currentThreshold
	}
	if currentDirection == "low" && candidateDirection == "low" {
		return candidateThreshold < currentThreshold
	}
	return false
}

func normalizeAlarmQuality(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "unknown", "uncertain":
		return "unknown"
	case "good":
		return "good"
	default:
		return "bad"
	}
}
