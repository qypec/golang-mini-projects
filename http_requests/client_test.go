package main

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	dataset = "dataset.xml"
)

type xmlUsers struct {
	Users []xmlUser `xml:"row"`
}

type xmlUser struct {
	ID        int    `xml:"id"`
	FirstName string `xml:"first_name"`
	LastName  string `xml:"last_name"`
	Age       int    `xml:"age"`
	About     string `xml:"about"`
	Gender    string `xml:"gender"`
}

type users []User

func (s users) Len() int      { return len(s) }
func (s users) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type ByIDAsc struct{ users }

func (s ByIDAsc) Less(i, j int) bool { return s.users[i].Id < s.users[j].Id }

type ByIDDesc struct{ users }

func (s ByIDDesc) Less(i, j int) bool { return s.users[i].Id > s.users[j].Id }

type ByAgeAsc struct{ users }

func (s ByAgeAsc) Less(i, j int) bool { return s.users[i].Age < s.users[j].Age }

type ByAgeDesc struct{ users }

func (s ByAgeDesc) Less(i, j int) bool { return s.users[i].Age > s.users[j].Age }

type ByNameAsc struct{ users }

func (s ByNameAsc) Less(i, j int) bool { return s.users[i].Name < s.users[j].Name }

type ByNameDesc struct{ users }

func (s ByNameDesc) Less(i, j int) bool { return s.users[i].Name > s.users[j].Name }

func getRequest(w http.ResponseWriter, r *http.Request) (*SearchRequest, error) {
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, err
	}

	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, err
	}

	orderBy, err := strconv.Atoi(r.URL.Query().Get("order_by"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, err
	}
	if orderBy != 1 && orderBy != 0 && orderBy != -1 {
		w.WriteHeader(http.StatusBadRequest)
		data, _ := json.Marshal(&SearchErrorResponse{"order_by has an invalid value"})
		w.Write(data)
		return nil, errors.New("order_by has an invalid value")
	}

	orderField := r.URL.Query().Get("order_field")
	if orderField != "Id" && orderField != "Age" && orderField != "Name" && orderField != "" {
		w.WriteHeader(http.StatusBadRequest)
		data, _ := json.Marshal(&SearchErrorResponse{"ErrorBadOrderField"})
		w.Write(data)
		return nil, errors.New(ErrorBadOrderField)
	}

	query := r.URL.Query().Get("query")

	return &SearchRequest{
		Limit:      limit,
		Offset:     offset,
		Query:      query,
		OrderField: orderField,
		OrderBy:    orderBy,
	}, nil
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	req, err := getRequest(w, r)
	if err != nil {
		return
	}

	file, err := os.Open(dataset)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	var x xmlUsers
	err = xml.Unmarshal(data, &x)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	sample := make([]User, 0)
	for _, user := range x.Users {
		name := user.FirstName + " " + user.LastName
		if strings.Contains(name, req.Query) || strings.Contains(user.About, req.Query) {
			sample = append(sample, User{
				Id:     user.ID,
				Name:   name,
				Age:    user.Age,
				About:  user.About,
				Gender: user.Gender,
			})
		}
	}

	if req.OrderBy != OrderByAsIs {
		switch req.OrderField {
		case "Id":
			if req.OrderBy == OrderByAsc {
				sort.Sort(ByIDAsc{sample})
			} else {
				sort.Sort(ByIDDesc{sample})
			}
		case "Age":
			if req.OrderBy == OrderByAsc {
				sort.Sort(ByAgeAsc{sample})
			} else {
				sort.Sort(ByAgeDesc{sample})
			}
		default: // "Name" || ""
			if req.OrderBy == OrderByAsc {
				sort.Sort(ByNameAsc{sample})
			} else {
				sort.Sort(ByNameDesc{sample})
			}
		}
	}

	json, _ := json.Marshal(sample)
	w.Write(json)
}

type TestCase struct {
	Request  *SearchRequest
	Expected *SearchResponse
	IsError  bool
}

func TestFindUsers(t *testing.T) {
	cases := []TestCase{
		/* 01 */ { // basic test
			Request: &SearchRequest{
				Limit:      30,
				Offset:     1,
				Query:      "Boyd",
				OrderField: "Id",
				OrderBy:    OrderByAsc,
			},
			Expected: &SearchResponse{
				Users: []User{
					{
						Id:     0,
						Name:   "Boyd Wolf",
						Age:    22,
						About:  "Nulla cillum enim voluptate consequat laborum esse excepteur occaecat commodo nostrud excepteur ut cupidatat. Occaecat minim incididunt ut proident ad sint nostrud ad laborum sint pariatur. Ut nulla commodo dolore officia. Consequat anim eiusmod amet commodo eiusmod deserunt culpa. Ea sit dolore nostrud cillum proident nisi mollit est Lorem pariatur. Lorem aute officia deserunt dolor nisi aliqua consequat nulla nostrud ipsum irure id deserunt dolore. Minim reprehenderit nulla exercitation labore ipsum.\n",
						Gender: "male",
					},
				},
				NextPage: false,
			},
			IsError: false,
		},
		/* 02 */ {
			Request: &SearchRequest{
				Limit: -10,
			},
			Expected: nil,
			IsError:  true,
		},
		/* 03 */ {
			Request: &SearchRequest{
				Offset: -10,
			},
			Expected: nil,
			IsError:  true,
		},
		/* 04 */ {
			Request: &SearchRequest{
				Limit:      1,
				Offset:     1,
				Query:      "Boyd",
				OrderField: "Error",
				OrderBy:    OrderByAsc,
			},
			Expected: nil,
			IsError:  true,
		},
		/* 05 */ {
			Request: &SearchRequest{
				Limit:      1,
				Offset:     1,
				Query:      "Boyd",
				OrderField: "Id",
				OrderBy:    100000,
			},
			Expected: nil,
			IsError:  true,
		},
		/* 06 */ { // limit == len(data)
			Request: &SearchRequest{
				Limit:      0,
				Offset:     1,
				Query:      "Boyd",
				OrderField: "Id",
				OrderBy:    OrderByAsc,
			},
			Expected: &SearchResponse{
				Users:    []User{},
				NextPage: true,
			},
			IsError: false,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	for caseNum, item := range cases {
		c := &SearchClient{
			URL: ts.URL,
		}
		actual, err := c.FindUsers(*item.Request)

		if err != nil && !item.IsError {
			t.Errorf("[%d] unexpected error: %#v", caseNum, err)
		}
		if err == nil && item.IsError {
			t.Errorf("[%d] expected error, got nil", caseNum)
		}
		if !reflect.DeepEqual(item.Expected, actual) {
			t.Errorf("[%d] wrong result, expected %#v, got %#v", caseNum, item.Expected, actual)
		}
	}
}

func SearchServerFatalError(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
}

func TestSearchServerFatalError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServerFatalError))
	defer ts.Close()

	c := &SearchClient{
		URL: ts.URL,
	}
	_, err := c.FindUsers(SearchRequest{})
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func SearchServerBadAcessToken(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusUnauthorized)
}

func TestBadAcessToken(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServerBadAcessToken))
	defer ts.Close()

	c := &SearchClient{
		URL: ts.URL,
	}
	_, err := c.FindUsers(SearchRequest{})
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func SearchServerInvalidJSON(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello error!"))
}

func SearchServerBadRequestInvalidJSON(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("hello error!"))
}

func TestInvalidJson(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServerInvalidJSON))
	c := &SearchClient{
		URL: ts.URL,
	}
	_, err := c.FindUsers(SearchRequest{})
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	ts.Close()

	ts = httptest.NewServer(http.HandlerFunc(SearchServerBadRequestInvalidJSON))
	defer ts.Close()
	c = &SearchClient{
		URL: ts.URL,
	}
	_, err = c.FindUsers(SearchRequest{})
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestNilUrl(t *testing.T) {
	c := &SearchClient{
		URL: "",
	}
	_, err := c.FindUsers(SearchRequest{})
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func SearchServerTimeout(w http.ResponseWriter, r *http.Request) {
	time.Sleep(time.Second)
}

func TestTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServerTimeout))
	c := &SearchClient{
		URL: ts.URL,
	}
	_, err := c.FindUsers(SearchRequest{})
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}
