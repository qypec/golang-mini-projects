package main

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

type TestCase struct {
	Request *SearchRequest
	Result  *SearchResponse
	IsError bool
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	order_field_param := r.URL.Query().Get("order_field")
	if order_field_param != "Id" && order_field_param != "Age" && order_field_param != "Name" && order_field_param != "" {
		w.Write([]byte("ErrorBadOrderField"))
	}
	// w.WriteHeader(http.StatusBadRequest)
}

func TestFindUsers(t *testing.T) {
	cases := []TestCase{
		TestCase{
			Request: &SearchRequest{
				Limit:      1,
				Offset:     1,
				Query:      "haha",
				OrderField: "blabla",
				OrderBy:    1,
			},
			Result:  nil,
			IsError: true,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	for caseNum, item := range cases {
		c := &SearchClient{
			URL: ts.URL,
		}
		result, err := c.FindUsers(*item.Request)

		if err != nil && !item.IsError {
			t.Errorf("[%d] unexpected error: %#v", caseNum, err)
		}
		if err == nil && item.IsError {
			t.Errorf("[%d] expected error, got nil", caseNum)
		}
		if !reflect.DeepEqual(item.Result, result) {
			t.Errorf("[%d] wrong result, expected %#v, got %#v", caseNum, item.Result, result)
		}
	}
}
