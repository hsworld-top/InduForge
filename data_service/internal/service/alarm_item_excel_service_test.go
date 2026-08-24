package service

import (
	"bytes"
	"context"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/indu-forge/data_service/internal/auth"
	"github.com/indu-forge/data_service/internal/repository"
)

func TestAlarmImportTemplateContainsStableSheetsAndColumns(t *testing.T) {
	projectID := "550e8400-e29b-41d4-a716-446655440000"
	service := NewAlarmItemService(repository.NewAlarmPolicyRepository(nil), repository.NewDataPointRepository(nil))
	workbook, err := service.BuildImportTemplate(context.Background(), &auth.Claims{UserID: "user", ProjectIDs: []string{projectID}}, projectID)
	if err != nil {
		t.Fatalf("BuildImportTemplate() error = %v", err)
	}
	file, err := excelize.OpenReader(bytes.NewReader(workbook.Content))
	if err != nil {
		t.Fatalf("open workbook: %v", err)
	}
	defer file.Close()
	if index, _ := file.GetSheetIndex(alarmExcelItemSheet); index < 0 {
		t.Fatal("missing 报警项 sheet")
	}
	if index, _ := file.GetSheetIndex(alarmExcelConditionSheet); index < 0 {
		t.Fatal("missing 报警条件 sheet")
	}
	if value, _ := file.GetCellValue(alarmExcelItemSheet, "A1"); value != "报警ID" {
		t.Fatalf("A1 = %q", value)
	}
	if value, _ := file.GetCellValue(alarmExcelConditionSheet, "F1"); value != "参数JSON" {
		t.Fatalf("F1 = %q", value)
	}
}

func TestAlarmImportRejectsFormulaCells(t *testing.T) {
	file := excelize.NewFile()
	defer file.Close()
	_ = file.SetSheetName(file.GetSheetName(0), alarmExcelItemSheet)
	_ = file.SetCellFormula(alarmExcelItemSheet, "A2", "=1+1")
	if err := rejectAlarmExcelFormulas(file, alarmExcelItemSheet); err == nil {
		t.Fatal("formula cell should be rejected")
	}
}
