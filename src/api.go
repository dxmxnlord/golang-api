package main

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

)

/*
	type Api - describes application
	@attributes:
		Db - MongoDb database instance
		pageSize - max size of each page for pagination
*/

type Api struct {
	Db *mongo.Database
	pageSize int
}

/*
	function Api.Run()
	@params: 
		addr - port
	@description: 
		Runs the application on localhost:addr 
*/

func (api *Api) Run(addr string) {
	log.Fatal(http.ListenAndServe(addr, nil))
}

/*
	function Api.Init()
	@params: 
		dbName - name of the database
	@description: 
		Initializes the application. This includes connecting to the mongoDb cluster, creating an instance of the mux router and setting the pageSize value
*/

func (api* Api) Init(dbName string){
	fmt.Println("Initializing....")

	var mongoURI string = "mongodb+srv://testuser:user123@cluster0.gltu3.gcp.mongodb.net/" + dbName + "?retryWrites=true&w=majority"
	
	// setup connection with mongoDb cluster
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil { log.Fatal(err) }
	api.Db = client.Database(dbName)
	fmt.Println("Connected to MongoDB!")

	// setup routes
	api.createRoutes()

	// setup pagesize
	api.pageSize = 2
}

/*
	function Api.createRoutes()
	@params: none
	@description:
		Add route handlers for the various Api routes. 
		Routes are either only POST or GET or both.
		Some routes take query parameters which is multiplexed through a middle handler `Api.getMeetingsHandler()`
*/

func (api* Api) createRoutes() {
	http.HandleFunc("/api/meeting", api.getMeeting)
	http.HandleFunc("/api/meetings", api.getMeetingsHandler)
}

/*
	function Api.getMeetingsHandler()
	@purpose:
		decide which handler to call for the path based on request type and params
	@params:
		http.ResponseWriter - for giving back a response
		http.Request - the original http request
	@description:
		If the method is a POST request, call the `Api.createMeeting()` handler
		If the method is GET:
			If the params are only email call the `Api.getMeetingsParticipant()` handler
			If the params are only start and end times call the `Api.getMeetings()` handler
		Else return an error response
*/

func (api* Api) getMeetingsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		api.createMeeting(w,r)
	} else if r.Method == "GET" {
		if r.FormValue("email") != "" && r.FormValue("start") == ""  && r.FormValue("end") == "" {
			api.getMeetingsParticipant(w,r)
		} else if r.FormValue("email") == "" && r.FormValue("start") != "" && r.FormValue("end") != ""{
			api.getMeetings(w,r)
		} else {
			errorResponse(w,http.StatusBadRequest,"Invalid query parameters")
		}
	} else {
		errorResponse(w,http.StatusMethodNotAllowed,"Invalid Request Method for this endpoint")
	}
}

/*
	function Api.createMeeting()
	@purpose:
		create new meeting
	@params:
		http.ResponseWriter - for giving back a response
		http.Request - the original http request
	@description:
		Decode the request body into a variable of type Meeting. 
		If any errors occur then we return an error status with a error status and error message. 
		Else we call the Meeting.createMeeting() function that inserts the new Meeting record into the collection.

		The helper functions are used for wrapping the response
*/

func (api* Api) createMeeting(w http.ResponseWriter, r *http.Request) {

	// decode the request body into a type Meeting variable and check for errors
	var meeting Meeting
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&meeting); err != nil {
		fmt.Println(err)
		errorResponse(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// insert into collection
	result, err := meeting.createMeeting(api.Db)

	if err != nil {
		fmt.Println(err)
		// failure
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// success
	jsonResponse(w, http.StatusCreated, result)
}

/*
	function Api.getMeetings()
	@purpose:
		get meetings within a range
	@params:
		http.ResponseWriter - for giving back a response
		http.Request - the original http request
	@description:
		Get the query params start and end from the url and check if they exist
		Parse the times into time.Time type inorder to compare with the times in the collection documents
		Check if pagination is to be done and retrieve the page count if so
		Call the getMeetings method to query the collection for meetings and pass the time range and page as params

		The helper functions are used for wrapping the response
*/

func (api *Api) getMeetings(w http.ResponseWriter, r *http.Request) {
	
	start := r.FormValue("start")
	end := r.FormValue("end")

	if start == "" || end == "" {
		errorResponse(w,http.StatusBadRequest,"Missing starting or Ending time")
		return
	}

	// parse times
	st_time, err := time.Parse("2006-01-02T15:04:05Z", start)
	if err != nil {
	    errorResponse(w, http.http.StatusBadRequest, err.Error())
	    return
	}
	en_time, err := time.Parse("2006-01-02T15:04:05Z", end)
	if err != nil {
	    errorResponse(w, http.http.StatusBadRequest, err.Error())
	    return
	}

	// check for pagination and get page
	page, err := strconv.Atoi(r.FormValue("page"))
	if err != nil && r.FormValue("page") != "" {
		errorResponse(w, http.http.StatusBadRequest, "Invalid page value")
		return
	}
	if r.FormValue("page") == "" {
		page = -1
	}

	// call aux method to query database
	meetings, err := getMeetings(api.Db,st_time,en_time,page,api.pageSize)
	if err != nil {
		errorResponse(w, http.http.StatusInternalServerError, err.Error())
		return
	}

	// success
	jsonResponse(w, http.StatusOK, meetings)

}

/*
	function Api.getMeetingsParticipant()
	@purpose:
		get meetings of a participant with email
	@params:
		http.ResponseWriter - for giving back a response
		http.Request - the original http request
	@description:
		Get the query param email from the url and check if t exists
		Check if pagination is to be done and retrieve the page count if so
		Call the getMeetingsParticipant method to query the collection for meetings and pass the email and page as params

		The helper functions are used for wrapping the response
*/

func (api *Api) getMeetingsParticipant(w http.ResponseWriter, r *http.Request) {

	// get url params
	email := r.FormValue("email")
	if email == "" {
		errorResponse(w,http.StatusBadRequest,"Missing email parameter")
		return
	}

	// check for pagination
	page, err := strconv.Atoi(r.FormValue("page"))
	if err != nil && r.FormValue("page") != "" {
		errorResponse(w, http.http.StatusBadRequest, "Invalid page value")
		return
	}
	if r.FormValue("page") == "" {
		page = -1
	}

	// call auxillary function to query database
	meetings, err := getMeetingsParticipant(api.Db,email,page,api.pageSize)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, meetings)
}

/*
	function Api.getMeeting()
	@purpose:
		get single meeting
	@params:
		http.ResponseWriter - for giving back a response
		http.Request - the original http request
	@description:
		Check if the method is GET, else return error response
		Get the query params from the url and check if id param exists
		Convert id parameter to mongoDb objectId
		Create the response Meeting object with Id initialized
		Call Meeting.getMeeting method to populate the response Meeting object

		The helper functions are used for wrapping the response
*/

func (api *Api) getMeeting(w http.ResponseWriter, r *http.Request) {

	if r.Method == "GET" {

		// get url params
		id := r.FormValue("id")
		if id == "" {
			errorResponse(w,http.StatusBadRequest,"No Id found")
			return 
		}
		mongoid, _ := primitive.ObjectIDFromHex(id)

		meeting := Meeting{ID:mongoid}

		// populate meeting object
		if err := meeting.getMeeting(api.Db); err != nil {
				switch err {
				case mongo.ErrNoDocuments:
					errorResponse(w, http.StatusNotFound, "Meeting not found")
				default:
					errorResponse(w, http.StatusInternalServerError, err.Error())
				}
				return
			}

		jsonResponse(w, http.StatusOK, meeting)
	} else {
		errorResponse(w,http.StatusMethodNotAllowed,"Invalid Request Method for this endpoint")
	}
}