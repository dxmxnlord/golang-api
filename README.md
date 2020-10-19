# Golang Meeting Rest API

This project was made as a submission for the Appointy Task 1 internship selection test.

+ Scoring:
	+ Completion Percentage
		+ Total working endpoints among the ones listed above.
		+ Meetings should not be overlapped i.e. one participant (uniquely identified by email) should not have 2 or more meetings with RSVP Yes with any overlap between their times.
	+ Quality of Code
		+ Reusability
		+ Consistency in naming variables, methods, functions, types
		+ Idiomatic i.e. in Go’s style
	+ Make the server thread safe i.e. it should not have any race conditions especially when two meetings are being booked simultaneously for the same participant with overlapping time.
	+ Add pagination to the list endpoint
	+ Add unit tests

## How to run

Follow these steps:

```bash
git clone golang-api
cd golang-api/src

go get gorilla/mux
go get go.mongodb.org/mongo-driver/mongo
go get go.mongodb.org/mongo-driver/bson

go run *.go
```

## Routes

|     Route     | Method |          Required Parameters         | Optional Parameters | Examples                                                                   | Purpose                      |
|:-------------:|--------|:------------------------------------:|---------------------|----------------------------------------------------------------------------|------------------------------|
| /api/meeting/ | GET    |            id - object id            | none                | /api/meeting?id=5f8cc2fe07f771d59746e199                                   | Get specific meeting         |
| /api/meetings | POST   |                 none                 | none                | /api/meetings                                                              | Create new meeting           |
| /api/meetings | GET    |           email - email id           | page - page number  | /api/meetings?email=rishi@gmail.com[&page=2]                               | Get meetings of participant  |
| /api/meetings | GET    | start,end - YYYY-MM-DD(T)HH:MM:SS(Z) | page - page number  | /api/meetings?start=2018-09-22T10:42:31Z&end=2018-09-22T19:42:31Z[&page=3] | Get meetings in a time range |

## Approach and Design

This was my first time using golang ( not my first time designing REST Apis and using MongoDb ), so I struggled initially to get a grasp on how APIs were designed in golang. The first thing I did was go through the golang tour in order to understand the syntax and data structures. After that I got quite confident in using the syntax to create structures and handlers. 

The next thing I did was find out common ways to design and API using golang. After some searching and analysing pros and cons, I decided on the following stack:

+ `gorilla/mux` for Routing the requests (highly flexible)
+ `mongo-driver/mongo` for interacting with mongoDb
+ `mongo-driver/bson` for using stored bson data
+ mongo cluster on Atlas

After that I designed the structure of the Meeting Record to be stored in the Mongo collection. The Meeting struct has an array of embedded Participant structs like an array of embedded documents in mongo. All the fields except the Id, CreatedAt, and Participants cannot be ommited, and the starting time, ending time, and created time are all of type `time.Time`. This standardizes all times and also makes querying and saving documents based on time easier since we can avoid conversions from RFC to mongo's ISODate as mongo recongnizes time.Time as a standard.

```golang
type Meeting struct {
	ID primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Title string `json:"title" bson:"title"`
	StartTime time.Time `json:"start_time" bson:"start_time"`
	EndTime time.Time `json:"end_time" bson:"end_time"`
	CreatedAt time.Time `json:"created_at,omitempty" bson:"created_at,omitempty"`
	Participants []Participant `json:"participants,omitempty" bson:"participants,omitempty"`
}

type Participant struct {
	Email string `json:"email" bson:"email"`
	Name string `json:"name" bson:"name"`
	RSVP string `json:"rsvp" bson:"rsvp"`
}
```

The next step was to design the API itself. While doing this, I had to do a lot of research and seaching stackoverflow for questions that arose and errors I got. Here are some of the references I used while designing the API and SO links that helped solve some problems I had.

+ [golang documentation](https://tour.golang.org/welcome/1)
+ [mux documentation](https://godoc.org/github.com/scyth/go-webproject/gwp/libs/gorilla/mux)
+ [parsing Dates in golang](https://golangcode.com/parsing-dates/)
+ [intoduction to the golang mongo-driver](https://www.mongodb.com/blog/post/mongodb-go-driver-tutorial)
+ [conversion between error and string](https://stackoverflow.com/questions/22170942/how-to-get-error-message-in-a-string-in-golang)
+ [running multiple go files](https://stackoverflow.com/questions/28081486/how-can-i-go-run-a-project-with-multiple-files-in-the-main-package)
+ [golang time and mongodb time](https://stackoverflow.com/questions/49657422/using-time-time-in-mongodb-record)

I spread the API code into 4 files:

+ main.go
	+ instantiates and runs the api
+ api.go
	+ has API type definition and handler methods
	+ creates router and routes
	+ maps routes to handler functions
	+ defines handler functions that take requests and return http responses with appropriate data by calling auxillary methods
+ meetings.go
	+ has Meeting and Participant definition
	+ has the auxillary handlers to query and insert into database collection
+ helpers.go
	+ defines helper functions for getting context, and wrappers for sending responses

### `api.go`

The Api type defines the router, database, and pageSize as its members. 

Upon calling the `Api.Init()` function, the connection with mongodb is set up and the database field is set. The router is initialized and the routes are created by calling the `Api.createRoutes()` function. Lastly the pageSize is set.

The `Api.Run()` function starts the server.

The `Api.createRoutes()` function creates the routes of the api and binds the routes to the appropriate handler functions

The `Api.createMeeting()` is the handler function for creating the meeting. It decodes the json body into a Meeting variable and calls the `Method.createMeeting()` function to insert the meeting record. Then it calls the helper functions to send the response.

The `Api.getMeetings()` is the handler function to query meetings between a start time and a end time and also takes an optional url parameter that paginates the response (if "page" is found on the url path). This function first converts both times into `time.Time` format and checks if pagination is to be done and then and then calls the `getMeetings()` method in `meeting.go` which queries the collection and returns an array of Meeting objects that are passed to the helper functions to give as response.

The `Api.getMeetingsParticipant()` is the handler function to query meetings of a particular participant email and also takes an optional url parameter that paginates the response (if "page" is found on the url path). It obtains the email from the params and checks for pagination. Then it calls the `getMeetingsParticipant` method in `meeting.go` which queries the collection and returns an array of Meeting objects that are passed to the helper functions to give as response.

The `Api.getMeeting()` is is the handler function to query a particular meeting. It takes the id param from the url, constructs a Meeting Object with the ID field set, and calls `Meeting.getMeeting()` to populate the other fields and passes the Meeting object to the helper functions to give as response.

### `meetings.go`

Defines the `Meeting` and `Participant` types discussed before. 

The `Meeting.createMeeting()` is the handler function to create a new meeting. When this function is called, all the fields in the meeting object are already populated from the request body. It resets the `CreateAt` field with the a new timestamp. Then it performs a few checks and if the checks fail, it returns errors. 

It checks if the start time is before the end time.

Then for every participant in the meeting, it checks if the emails follow the proper format and checks if the rsvp is of the appropriate choices. 

For any participant, if the rsvp is yes, it checks for overlapping meetings. For this it queries the collection with this filter. If any meetings are present in the result then there exists an overlapping meeting and an error is returned. Else the object is created.

```golang
bson.M{
	// participant is going
	"participants": bson.D{
		{"email", participant.Email},
		{"name", participant.Name},
		{"rsvp", "Yes"},
	},
 	"$or" : bson.A{
 		// meeting starting and ending between another meeting
 		bson.M{
 			"start_time" : bson.M{
 				"$lte" : meeting.StartTime,
 			},
 			"end_time" : bson.M{
 				"$gte": meeting.EndTime,
 			},
 		},
 		// meeting starting before another but not getting over before start
 		bson.M{
 			"start_time" : bson.M{
 				"$gte" : meeting.StartTime,
 				"$lt": meeting.EndTime,
 			},
 			"end_time" : bson.M{
 				"$gte": meeting.StartTime,
 			},
 		},
 		// meeting starting during another meeting
 		bson.M{
 			"start_time" : bson.M{
 				"$gte" : meeting.StartTime,
 			},
 			"end_time" : bson.M{
 				"$gt": meeting.StartTime,
 				"$lte": meeting.EndTime,
 			},
 		},
	}, 
}
```

The `Meeting.getMeeting()` handler function gets a meeting from the collection and populates the Meeting instance with the fields. It uses the filter

```golang
bson.M{
	"_id": meeting.ID
}
```

The `getMeetings()` queries the collection for meetings within a time range. It uses the below filter and then returns the Meetings that lie in the range after paginating if opted for.

```golang
bson.M{
	"start_time" : bson.M{
		"$gte" : start_time
	}, 
	"end_time" : bson.M{
		"$lte" : end_time
	}
}
```

```golang
// Pagination
if page == -1 {
	// pagination is not chosen
	return meetings,nil
} else {
	// full range of page availible in result
	if len(meetings) >=  pageSize * page {
		return meetings[(page-1) * pageSize : page * pageSize],nil
	}
	// partial page availible in result
	else if len(meetings) <  pageSize * page && len(meetings) > (page-1)* pageSize { 
		return meetings[(page-1) * pageSize : len(meetings)],nil
	}
	// page out of bounds
	else {
		return nil,nil
	}
}
```

The `getMeetingsParticipant()` does almost the same job as the `getMeeting()` except it queries based on the email field in the Participant and then paginates. It uses this filter.

```golang
bson.M{
	"participants.email": email
}
```

### `helpers.go`

The `getContext()` returns a new context with a timeout.

The `jsonResponse()` sends a response after marshalling the data send to it with an appropriate status.

The `errorResponse()` sends an error message as a response.