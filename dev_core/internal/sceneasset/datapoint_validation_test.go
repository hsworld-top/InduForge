package sceneasset

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type validationRoundTripper func(*http.Request) (*http.Response, error)

func (roundTrip validationRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTrip(request)
}

func TestValidateDatapointsChecksCapabilitiesAndType(t *testing.T) {
	client := &http.Client{Transport: validationRoundTripper(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("authorization was not forwarded")
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(
			`{"code":0,"msg":"ok","data":{"datapoints":[{"path":"line.temperature","dataType":"float64","status":"active","capabilities":{"get":true,"sub":true,"set":true}}],"pagination":{"totalPages":1}}}`,
		))}, nil
	})}

	handler := &Handler{dataServiceURL: "http://data-service.test", httpClient: client}
	request, err := http.NewRequest(http.MethodPost, "http://dev-core.test/commit", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer test-token")
	err = handler.validateDatapoints(request, "project-a", []DatapointRequirement{{
		Path: "line.temperature", ValueType: "number", Get: true, Sub: true, Set: true,
	}})
	if err != nil {
		t.Fatalf("expected valid datapoint binding: %v", err)
	}
}

func TestDatapointTypesCompatible(t *testing.T) {
	for _, item := range []struct{ expected, actual string }{
		{"number", "float64"}, {"number", "int"}, {"integer", "int64"}, {"boolean", "bool"}, {"string", "text"}, {"object", "json"},
	} {
		if !datapointTypesCompatible(item.expected, item.actual) {
			t.Fatalf("expected %s and %s to be compatible", item.expected, item.actual)
		}
	}
	if datapointTypesCompatible("boolean", "float64") {
		t.Fatal("boolean and float64 must be incompatible")
	}
}
