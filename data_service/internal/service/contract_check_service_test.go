package service

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/indu-forge/data_service/internal/auth"
	"github.com/indu-forge/data_service/internal/repository"
)

func TestLoadAllContractCheckPages(t *testing.T) {
	const total = 1201
	calls := 0
	records, err := loadAllContractCheckPages(100, func(page, pageSize int) ([]int, int, error) {
		calls++
		start := (page - 1) * pageSize
		if start >= total {
			return []int{}, total, nil
		}
		end := min(start+pageSize, total)
		pageRecords := make([]int, 0, end-start)
		for index := start; index < end; index++ {
			pageRecords = append(pageRecords, index)
		}
		return pageRecords, total, nil
	})
	if err != nil {
		t.Fatalf("loadAllContractCheckPages() error = %v", err)
	}
	if len(records) != total {
		t.Fatalf("len(records) = %d, want %d", len(records), total)
	}
	if calls != 13 {
		t.Fatalf("calls = %d, want 13", calls)
	}
	if records[0] != 0 || records[len(records)-1] != total-1 {
		t.Fatalf("records boundary = %d..%d", records[0], records[len(records)-1])
	}

	wantErr := fmt.Errorf("page failed")
	_, err = loadAllContractCheckPages(100, func(page, _ int) ([]int, int, error) {
		if page == 2 {
			return nil, total, wantErr
		}
		return make([]int, 100), total, nil
	})
	if err != wantErr {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}

func TestToContractCheckRun(t *testing.T) {
	summaryJSON := json.RawMessage(`{"passed":5,"warning":2,"failed":1}`)
	resultJSON := json.RawMessage(`{"status":"passed","projectId":"p1","summary":{"passed":5}}`)

	record := repository.ContractCheckRunRecord{
		ID:         10,
		ProjectID:  "proj-1",
		Scope:      "project",
		ObjectType: ptr("datapoint"),
		ObjectID:   ptr("dp-1"),
		Status:     "warning",
		Summary:    summaryJSON,
		Result:     resultJSON,
		CreatedBy:  ptr("user-1"),
		CreatedAt:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	run, err := toContractCheckRun(record)
	if err != nil {
		t.Fatalf("toContractCheckRun() error = %v", err)
	}

	if run.ID != 10 {
		t.Errorf("ID = %v, want 10", run.ID)
	}
	if run.Status != "warning" {
		t.Errorf("Status = %v, want warning", run.Status)
	}
	if run.Summary == nil {
		t.Fatal("Summary is nil")
	}
	if run.Summary["passed"] != 5 || run.Summary["warning"] != 2 || run.Summary["failed"] != 1 {
		t.Errorf("Summary = %v, want passed=5,warning=2,failed=1", run.Summary)
	}
	if run.Result == nil || string(run.Result) != string(resultJSON) {
		t.Errorf("Result = %v, want %v", string(run.Result), string(resultJSON))
	}
}

func TestBuildContractCheckResult(t *testing.T) {
	tests := []struct {
		name        string
		items       []ContractCheckResultItem
		wantStatus  string
		wantSummary map[string]int
	}{
		{
			name:        "all passed",
			items:       []ContractCheckResultItem{{Status: "passed"}, {Status: "passed"}},
			wantStatus:  "passed",
			wantSummary: map[string]int{"passed": 2, "warning": 0, "pending": 0, "failed": 0},
		},
		{
			name:        "failed takes precedence",
			items:       []ContractCheckResultItem{{Status: "passed"}, {Status: "failed"}, {Status: "warning"}},
			wantStatus:  "failed",
			wantSummary: map[string]int{"passed": 1, "warning": 1, "pending": 0, "failed": 1},
		},
		{
			name:        "warning without failed",
			items:       []ContractCheckResultItem{{Status: "passed"}, {Status: "warning"}},
			wantStatus:  "warning",
			wantSummary: map[string]int{"passed": 1, "warning": 1, "pending": 0, "failed": 0},
		},
		{
			name:        "pending without failed/warning",
			items:       []ContractCheckResultItem{{Status: "passed"}, {Status: "pending"}},
			wantStatus:  "pending",
			wantSummary: map[string]int{"passed": 1, "warning": 0, "pending": 1, "failed": 0},
		},
		{
			name:        "empty list",
			items:       []ContractCheckResultItem{},
			wantStatus:  "passed",
			wantSummary: map[string]int{"passed": 1, "warning": 0, "pending": 0, "failed": 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildContractCheckResult("p1", "project", tt.items)
			if result.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v", result.Status, tt.wantStatus)
			}
			for k, v := range tt.wantSummary {
				if result.Summary[k] != v {
					t.Errorf("Summary[%s] = %v, want %v", k, result.Summary[k], v)
				}
			}
		})
	}
}

func TestContractCheckServiceListRuns_NilChecks(t *testing.T) {
	claims := &auth.Claims{UserID: "user-1", ProjectIDs: []string{"550e8400-e29b-41d4-a716-446655440000"}}
	s := &ContractCheckService{
		datapoints: &repository.DataPointRepository{},
		compute:    &repository.ComputeRepository{},
		alarms:     &repository.AlarmRepository{},
		checks:     nil,
	}

	result, err := s.ListRuns(context.Background(), claims, "550e8400-e29b-41d4-a716-446655440000", 1, 20)
	if err != nil {
		t.Fatalf("ListRuns() error = %v", err)
	}
	if result == nil {
		t.Fatal("ListRuns() returned nil result")
	}
	if len(result.Runs) != 0 {
		t.Errorf("Runs = %v, want empty", result.Runs)
	}
}

func ptr(s string) *string {
	return &s
}
