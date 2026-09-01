package hostd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPIRejectsUnknownFieldsAndNonDeclarativeOperation(t *testing.T) {
	manager, _ := preparedManager(t)
	api, err := NewAPI(manager)
	if err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{
		`{"schemaVersion":"induforge.cluster-plan.v1","unknown":"value"}`,
		`{"schemaVersion":"induforge.cluster-plan.v1","generation":1,"clusterId":"` + testEnvironmentID + `","nodeId":"` + testNodeID + `","operation":"shell","k3sVersion":"` + K3sVersion + `","token":"` + testToken + `","nodeIp":"172.16.125.129","dataDir":"/var/lib/induforge/k3s"}`,
	} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/v1/cluster/apply", strings.NewReader(body))
		api.Handler().ServeHTTP(response, request)
		if response.Code < 400 {
			t.Fatalf("expected rejection, status=%d body=%s", response.Code, response.Body.String())
		}
	}
}
