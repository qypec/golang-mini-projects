package main

import (
	"encoding/json"
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

const (
	dataFileName = "dataset.xml"
)

type users struct {
	Users []xmlUser `xml:"row"`
}

type xmlUser struct {
	Id        int    `xml:"id"`
	FirstName string `xml:"first_name"`
	LastName  string `xml:"last_name"`
	Age       int    `xml:"age"`
	About     string `xml:"about"`
	Gender    string `xml:"gender"`
}

type TestCase struct {
	Request *SearchRequest
	Result  *SearchResponse
	IsError bool
}

func getRequest(w http.ResponseWriter, r *http.Request) (*SearchRequest, error) {
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("limit is not a number"))
		return nil, err
	}
	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("offset is not a number"))
		return nil, err
	}
	orderBy, err := strconv.Atoi(r.URL.Query().Get("order_by"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("order_by is not a number"))
		return nil, err
	}
	return &SearchRequest{
		Limit:      limit,
		Offset:     offset,
		Query:      r.URL.Query().Get("query"),
		OrderField: r.URL.Query().Get("order_field"),
		OrderBy:    orderBy,
	}, nil
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	req, err := getRequest(w, r)
	if err != nil {
		return
	}

	file, err := os.Open(dataFileName)
	if err != nil {
		panic(err) // fix
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err) // fix
	}

	var u users
	err = xml.Unmarshal(data, &u)
	if err != nil {
		panic(err) // fix
	}

	sample := make([]User, 0)
	for _, user := range u.Users {
		name := user.FirstName + " " + user.LastName
		if strings.Contains(name, req.Query) || strings.Contains(user.About, req.Query) {
			sample = append(sample, User{
				Id:     user.Id,
				Name:   name,
				Age:    user.Age,
				About:  user.About,
				Gender: user.Gender,
			})
		}
	}

	// sort.Slice(sample, sortRules(req))

	json, _ := json.Marshal(sample) // err ??
	w.Write(json)
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
			Result:  &SearchResponse{},
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
