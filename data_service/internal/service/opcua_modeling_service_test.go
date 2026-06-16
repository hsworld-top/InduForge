package service

import (
	"reflect"
	"strings"
	"testing"
)

func TestNormalizeOpcuaImportRowsGeneratesUniqueCodes(t *testing.T) {
	existingCodes := map[string]struct{}{"speed": {}}
	input := []ImportOpcuaNodeInput{
		{Name: "Speed", NodeID: "ns=2;s=Line1.Speed", DataType: "Double"},
		{Name: "Speed", NodeID: "ns=2;s=Line1.Speed2", DataType: "Double"},
	}

	rows, err := normalizeOpcuaImportRows(input, existingCodes)
	if err != nil {
		t.Fatalf("normalize import rows: %v", err)
	}

	got := []string{rows[0].Code, rows[1].Code}
	want := []string{"speed_2", "speed_3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("codes = %#v, want %#v", got, want)
	}
}

func TestNormalizeOpcuaImportRowsRejectsDuplicateNodeID(t *testing.T) {
	input := []ImportOpcuaNodeInput{
		{Name: "Speed", NodeID: "ns=2;s=Line1.Speed", DataType: "Double"},
		{Name: "Speed Copy", NodeID: "ns=2;s=Line1.Speed", DataType: "Double"},
	}

	_, err := normalizeOpcuaImportRows(input, nil)
	if err == nil {
		t.Fatal("expected duplicate NodeId error")
	}
	if !strings.Contains(err.Error(), "批量导入包含重复 NodeId") {
		t.Fatalf("error = %v", err)
	}
}

func TestNormalizeOpcuaImportRowsRejectsTooManyRows(t *testing.T) {
	input := make([]ImportOpcuaNodeInput, maxOpcuaBatchImportNodes+1)
	for index := range input {
		input[index] = ImportOpcuaNodeInput{
			Name:     "Node",
			NodeID:   "ns=2;s=Node" + string(rune(index+65)),
			DataType: "Double",
		}
	}

	_, err := normalizeOpcuaImportRows(input, nil)
	if err == nil {
		t.Fatal("expected max rows error")
	}
	if !strings.Contains(err.Error(), "单次最多导入") {
		t.Fatalf("error = %v", err)
	}
}
