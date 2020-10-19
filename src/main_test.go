package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// application object

var api Api

// base

func TestMain(mainTest *testing.M) {
	api = Api{}
	api.Init("appointy-api-test")
	code := mainTest.Run()
	dropDatabase()
	os.Exit(code)
}

// helper functions

func dropDatabase() {
	ctx := getContext(10)
	err := api.Db.Drop(ctx)
	if err != nil {
		log.Fatalf("Could not drop database: %v", err)
	}
}

func newReq(req *http.Request) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	api.Router.ServeHTTP(recorder, req)

	return recorder
}

func checkResStatus(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
	}
}

// actual tests

func TestGetMeetingThatDoesNotExist(t *testing.T) {
	dropDatabase()

	req, _ := http.NewRequest("GET", "/api/meeting?id=12314", nil)
	response := newReq(req)

	checkResStatus(t, http.StatusNotFound, response.Code)

	var data map[string]string
	json.Unmarshal(response.Body.Bytes(), &data)
	if data["error"] != "Meeting not found" {
		t.Errorf("Expected error 'Meeting not found', but instead '%s' was returned", data["error"])
	}
}

func TestCreateMeeting(t *testing.T) {
	dropDatabase()

	payload := []byte(`{
	    "title" : "meeting1",
	    "start_time": "2020-10-19T13:00:00Z",
	    "end_time": "2020-10-19T15:00:00Z",
	    "participants" : [
	        {
	            "name": "p1",
	            "email": "p1@gmail.com",
	            "rsvp": "No"
	        },
	        {
	            "name": "p2",
	            "email": "p2@gmail.com",
	            "rsvp": "Yes"
	        }
	    ]
	}`)

	req, _ := http.NewRequest("POST", "/api/meetings", bytes.NewBuffer(payload))
	response := newReq(req)

	checkResStatus(t, http.StatusCreated, response.Code)
}

func TestCreateMeetingRepeatedEmail(t *testing.T){
	dropDatabase()

	payload := []byte(`{
	    "title" : "meeting1",
	    "start_time": "2020-10-19T13:00:00Z",
	    "end_time": "2020-10-19T15:00:00Z",
	    "participants" : [
	        {
	            "name": "p1",
	            "email": "p1@gmail.com",
	            "rsvp": "No"
	        },
	        {
	            "name": "p1",
	            "email": "p1@gmail.com",
	            "rsvp": "Yes"
	        }
	    ]
	}`)

	req, _ := http.NewRequest("POST", "/api/meetings", bytes.NewBuffer(payload))
	response := newReq(req)

	checkResStatus(t, http.StatusBadRequest, response.Code)

	var data map[string]string
	json.Unmarshal(response.Body.Bytes(), &data)
	if data["error"] != "Repeated email found" {
		t.Errorf("Expected error 'Repeated email found', but instead '%s' was returned", data["error"])
	}
}

func TestCreateMeetingGreaterStartTime(t *testing.T){
	dropDatabase()

	payload := []byte(`{
	    "title" : "meeting1",
	    "start_time": "2020-10-19T16:00:00Z",
	    "end_time": "2020-10-19T15:00:00Z",
	    "participants" : [
	        {
	            "name": "p1",
	            "email": "p1@gmail.com",
	            "rsvp": "No"
	        },
	        {
	            "name": "p2",
	            "email": "p2@gmail.com",
	            "rsvp": "Yes"
	        }
	    ]
	}`)

	req, _ := http.NewRequest("POST", "/api/meetings", bytes.NewBuffer(payload))
	response := newReq(req)

	checkResStatus(t, http.StatusBadRequest, response.Code)

	var data map[string]string
	json.Unmarshal(response.Body.Bytes(), &data)
	if data["error"] != "Start time is not before End time" {
		t.Errorf("Expected error 'Start time is not before End time', but instead '%s' was returned", data["error"])
	}
}

func TestCreateMeetingInvalidEmail(t *testing.T){
	dropDatabase()

	payload := []byte(`{
	    "title" : "meeting1",
	    "start_time": "2020-10-19T15:00:00Z",
	    "end_time": "2020-10-19T16:00:00Z",
	    "participants" : [
	        {
	            "name": "p1",
	            "email": "p1gmail.com",
	            "rsvp": "No"
	        },
	        {
	            "name": "p2",
	            "email": "p2@gmail.com",
	            "rsvp": "Yes"
	        }
	    ]
	}`)

	req, _ := http.NewRequest("POST", "/api/meetings", bytes.NewBuffer(payload))
	response := newReq(req)

	checkResStatus(t, http.StatusBadRequest, response.Code)

	var data map[string]string
	json.Unmarshal(response.Body.Bytes(), &data)
	if data["error"] != "Invalid email in participant list" {
		t.Errorf("Expected error 'Invalid email in participant list', but instead '%s' was returned", data["error"])
	}
}

func TestCreateMeetingParticipantInOverlappingMeetingsWhereMeetingStartsBefore(t *testing.T){
	dropDatabase()

	payload := []byte(`{
	    "title" : "meeting1",
	    "start_time": "2020-10-19T15:00:00Z",
	    "end_time": "2020-10-19T17:00:00Z",
	    "participants" : [
	        {
	            "name": "p1",
	            "email": "p1@gmail.com",
	            "rsvp": "Yes"
	        }
	    ]
	}`)

	req, _ := http.NewRequest("POST", "/api/meetings", bytes.NewBuffer(payload))
	newReq(req)

	payload2 := []byte(`{
	    "title" : "meeting2",
	    "start_time": "2020-10-19T14:00:00Z",
	    "end_time": "2020-10-19T16:00:00Z",
	    "participants" : [
	        {
	            "name": "p1",
	            "email": "p1@gmail.com",
	            "rsvp": "Yes"
	        }
	    ]
	}`)

	req2, _ := http.NewRequest("POST", "/api/meetings", bytes.NewBuffer(payload2))
	response := newReq(req2)

	checkResStatus(t, http.StatusBadRequest, response.Code)

	var data map[string]string
	json.Unmarshal(response.Body.Bytes(), &data)
	if data["error"] != "Overlapping meeting for email p1@gmail.com" {
		t.Errorf("Expected error 'Overlapping meeting for email p1@gmail.com', but instead '%s' was returned", data["error"])
	}
}

func TestCreateMeetingParticipantInOverlappingMeetingsWhereMeetingEndsAfter(t *testing.T){
	dropDatabase()

	payload := []byte(`{
	    "title" : "meeting1",
	    "start_time": "2020-10-19T15:00:00Z",
	    "end_time": "2020-10-19T17:00:00Z",
	    "participants" : [
	        {
	            "name": "p1",
	            "email": "p1@gmail.com",
	            "rsvp": "Yes"
	        }
	    ]
	}`)

	req, _ := http.NewRequest("POST", "/api/meetings", bytes.NewBuffer(payload))
	newReq(req)

	payload2 := []byte(`{
	    "title" : "meeting1",
	    "start_time": "2020-10-19T16:00:00Z",
	    "end_time": "2020-10-19T18:00:00Z",
	    "participants" : [
	        {
	            "name": "p1",
	            "email": "p1@gmail.com",
	            "rsvp": "Yes"
	        }
	    ]
	}`)

	req2, _ := http.NewRequest("POST", "/api/meetings", bytes.NewBuffer(payload2))
	response := newReq(req2)

	checkResStatus(t, http.StatusBadRequest, response.Code)

	var data map[string]string
	json.Unmarshal(response.Body.Bytes(), &data)
	if data["error"] != "Overlapping meeting for email p1@gmail.com" {
		t.Errorf("Expected error 'Overlapping meeting for email p1@gmail.com', but instead '%s' was returned", data["error"])
	}
}

func TestCreateMeetingParticipantInOverlappingMeetingsWhereMeetingInBetween(t *testing.T){
	dropDatabase()

	payload := []byte(`{
	    "title" : "meeting1",
	    "start_time": "2020-10-19T15:00:00Z",
	    "end_time": "2020-10-19T18:00:00Z",
	    "participants" : [
	        {
	            "name": "p1",
	            "email": "p1@gmail.com",
	            "rsvp": "Yes"
	        }
	    ]
	}`)

	req, _ := http.NewRequest("POST", "/api/meetings", bytes.NewBuffer(payload))
	newReq(req)

	payload2 := []byte(`{
	    "title" : "meeting1",
	    "start_time": "2020-10-19T16:00:00Z",
	    "end_time": "2020-10-19T17:00:00Z",
	    "participants" : [
	        {
	            "name": "p1",
	            "email": "p1@gmail.com",
	            "rsvp": "Yes"
	        }
	    ]
	}`)

	req2, _ := http.NewRequest("POST", "/api/meetings", bytes.NewBuffer(payload2))
	response := newReq(req2)

	checkResStatus(t, http.StatusBadRequest, response.Code)

	var data map[string]string
	json.Unmarshal(response.Body.Bytes(), &data)
	if data["error"] != "Overlapping meeting for email p1@gmail.com" {
		t.Errorf("Expected error 'Overlapping meeting for email p1@gmail.com', but instead '%s' was returned", data["error"])
	}
}