package service

import (
	"bytes"
	"context"
	"testing"

	"github.com/indu-forge/data_service/internal/repository"
	"github.com/xuri/excelize/v2"
)

func TestCollectorImportPreviewParsesSchemaAddressColumns(t *testing.T) {
	connectionID := "550e8400-e29b-41d4-a716-446655440002"
	pointStore := &fakeCollectorPointStore{connection: repository.CollectorConnectionRecord{ID: connectionID, ProjectID: "550e8400-e29b-41d4-a716-446655440000", DriverID: "opcua.standard", SchemaVersion: 1}}
	pointService := newRepositoryCollectorPointService(t, pointStore)
	importStore := &fakeCollectorImportStore{}
	service, err := NewCollectorImportService(importStore, pointService, pointService.catalog)
	if err != nil {
		t.Fatal(err)
	}
	content := []byte("groupPath,name,dataType,elementCount,address.nodeId\n设备一,温度,float32,1,ns=2;s=Temperature\n")
	preview, err := service.Preview(context.Background(), pointStore.connection.ProjectID, connectionID, "550e8400-e29b-41d4-a716-446655440001", "points.csv", content, 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if preview.ValidRows != 1 || preview.Candidates[0].AddressText != "ns=2;s=Temperature" || importStore.session.Candidates[0].GroupPath != "设备一" {
		t.Fatalf("unexpected preview: %#v", preview)
	}
}

func TestCollectorImportTemplateCreatesXLSX(t *testing.T) {
	pointStore := &fakeCollectorPointStore{}
	pointService := newRepositoryCollectorPointService(t, pointStore)
	service, err := NewCollectorImportService(&fakeCollectorImportStore{}, pointService, pointService.catalog)
	if err != nil {
		t.Fatal(err)
	}
	template, err := service.BuildTemplate("opcua.standard", "xlsx")
	if err != nil {
		t.Fatal(err)
	}
	if len(template.Content) < 100 || template.ContentType == "" {
		t.Fatalf("invalid template: %#v", template)
	}
	file, err := excelize.OpenReader(bytes.NewReader(template.Content))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	rows, err := file.GetRows(file.GetSheetList()[0])
	if err != nil {
		t.Fatal(err)
	}
	headers := map[string]bool{}
	for _, header := range rows[0] {
		headers[header] = true
	}
	for _, expected := range []string{"acquisitionMode", "acquisitionOverrides"} {
		if !headers[expected] {
			t.Fatalf("模板缺少字段 %s: %#v", expected, rows[0])
		}
	}
	if headers["acquisition"] {
		t.Fatalf("模板仍包含已废弃字段 acquisition: %#v", rows[0])
	}
}

type fakeCollectorImportStore struct {
	session repository.CollectorImportSessionRecord
}

func (f *fakeCollectorImportStore) CreateImportSession(_ context.Context, params repository.CreateCollectorImportSessionParams) (*repository.CollectorImportSessionRecord, error) {
	f.session = repository.CollectorImportSessionRecord{ID: params.ID, ProjectID: params.ProjectID, ConnectionID: params.ConnectionID, DriverID: params.DriverID, Status: "preview", Candidates: params.Candidates, Errors: params.Errors, TotalRows: params.TotalRows, ValidRows: len(params.Candidates), ExpiresAt: params.ExpiresAt}
	return &f.session, nil
}
func (f *fakeCollectorImportStore) GetImportSession(context.Context, string, string, string) (*repository.CollectorImportSessionRecord, error) {
	return &f.session, nil
}
func (f *fakeCollectorImportStore) CommitImportSession(context.Context, string, string, string) ([]repository.CollectorPointRecord, error) {
	return nil, nil
}
