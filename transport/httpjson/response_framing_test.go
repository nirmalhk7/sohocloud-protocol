package httpjson_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/NTARI-RAND/sohocloud-protocol/employment"
	"github.com/NTARI-RAND/sohocloud-protocol/fees"
	"github.com/NTARI-RAND/sohocloud-protocol/transport/httpjson"
)

func TestHandlerEncodesOneSuccessfulDocument(t *testing.T) {
	at := time.Unix(1_700_000_000, 0).UTC()
	handler := httpjson.Handler(&recordingCoordinator{
		jobs: []employment.Assignment{{JobID: "job", OfferedAt: at}},
		fees: fees.FeeDeclaration{EffectiveAt: at},
	})
	for _, path := range []string{"/v0/jobs?node_id=node", "/v0/fees"} {
		t.Run(path, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
			if response.Code != http.StatusOK || !json.Valid(response.Body.Bytes()) {
				t.Fatalf("status=%d body=%q; want one complete JSON value", response.Code, response.Body)
			}
		})
	}
}

// Removing EOF validation would allow a second value or junk.
func TestClientCompleteResponse(t *testing.T) {
	for _, method := range []string{"jobs", "fees"} {
		value := "[]"
		if method == "fees" {
			value = "{}"
		}
		for _, suffix := range []string{"", " \t\r\n", " {}", " []", " null", " junk", "\v"} {
			t.Run(method+"/"+suffix, func(t *testing.T) {
				body := &trackedBody{Reader: strings.NewReader(value + suffix)}
				err := invokeRequest(responseClient(body, -1), context.Background(), method)
				valid := suffix == "" || suffix == " \t\r\n"
				if (err == nil) != valid {
					t.Fatalf("error = %v; valid = %v", err, valid)
				}
				if !body.closed {
					t.Fatal("response not closed")
				}
			})
		}
	}
}

func TestClientDoesNotReturnPartialResponse(t *testing.T) {
	for _, limit := range []int64{0, 1024} {
		body := &trackedBody{Reader: strings.NewReader(`[{"JobID":"not-an-assignment"}] {}`)}
		client := responseClient(body, -1)
		client.MaxResponseBytes = limit
		jobs, err := client.PollJobs(context.Background(), "node")
		if err == nil || jobs != nil {
			t.Fatalf("jobs=%v error=%v; want nil result and error", jobs, err)
		}
		body = &trackedBody{Reader: strings.NewReader(`{"Seq":7} {}`)}
		client = responseClient(body, -1)
		client.MaxResponseBytes = limit
		fee, err := client.Fees(context.Background())
		if err == nil || fee.Seq != 0 {
			t.Fatalf("fee=%v error=%v; want zero result and error", fee, err)
		}
	}
}

// Invalid configuration must fail before a request reaches its destination.
func TestClientRejectsNegativeBudget(t *testing.T) {
	called := false
	client := &httpjson.Client{BaseURL: "https://coordinator.example.test", MaxResponseBytes: -1,
		HTTP: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { called = true; return nil, io.EOF })}}
	if err := invokeRequest(client, context.Background(), "jobs"); err == nil {
		t.Fatal("negative budget accepted")
	}
	if called {
		t.Fatal("negative budget reached transport")
	}
}

func TestClientResponseVectors(t *testing.T) {
	data, err := os.ReadFile("../../testdata/response_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name, Method, Body string
		Limit              int64
		Accept             bool
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			body := &trackedBody{Reader: strings.NewReader(tc.Body)}
			client := responseClient(body, -1)
			client.MaxResponseBytes = tc.Limit
			err := invokeRequest(client, context.Background(), tc.Method)
			if (err == nil) != tc.Accept {
				t.Fatalf("error=%v; accept=%v", err, tc.Accept)
			}
		})
	}
}
