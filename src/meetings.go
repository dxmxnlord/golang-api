package main

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/bson"
	"time"
	"errors"
	"fmt"
	"regexp"
)

// Object Types

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

/*
	function Meeting.()
	@purpose
	@params:
	@description:
	@return:
*/

/*
	function Meeting.createMeeting()
	@purpose:
		inserts meeting object into collection
	@params:
		db - mongoDb database instance
	@description:
		get collection and context
		set the CreatedAt timestamp to current time
		perform checks:
			ensure starttime is before the end time
			for each participant in the meeting:
				ensure email is used only once
				ensure email is in right format
				ensure meetings are not be overlapped for participant (if rsvp is yes)
				ensure rsvp is in chosen values
		call collection.InsertOne() to insert meeting record
	@return:
		mongo.InsertOneResult - InsertOneResult | nil
		error - nil | error
*/

func (meeting *Meeting) createMeeting (db *mongo.Database) (*mongo.InsertOneResult, error) {

	// get collection and context
	collection := db.Collection("meetings")
	ctx := getContext(10)

	// set the CreatedAt timestamp to current time
	meeting.CreatedAt = time.Now()

	// ensure starttime is before the end time
	if meeting.EndTime.Before(meeting.StartTime) == true {
		return nil, errors.New("Start time is not before End time")
	}

	var emails map[string]bool = map[string]bool{}

	for i:=0;i<len(meeting.Participants);i++ {
		// get a participant
		participant := meeting.Participants[i]

		// ensure only one email of each participant
		_,ok := emails[participant.Email]
		if ok == true {
			return nil, errors.New("Repeated email found")
		} else {
			emails[participant.Email] = true;
		}

		// check email format
		match, _ := regexp.MatchString("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$", participant.Email)
		if match == false {
			return nil, errors.New("Invalid email in participant list")
		}

		// if rsvp is Yes, check for overlapping meetings
		if participant.RSVP == "Yes" {
			filter := bson.M{
				// participant is going for meeting
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
			 				"$lte" : meeting.StartTime,
			 			},
			 			"end_time" : bson.M{
			 				"$gt": meeting.StartTime,
			 				"$lte": meeting.EndTime,
			 			},
			 		},
				}, 
			}

			// find meetings with overlapping conditions
			cursor,err := collection.Find(ctx,filter)
			var meetingsCheck []bson.M
			if err = cursor.All(ctx, &meetingsCheck); err != nil {
			    return nil,err
			}
			// if returned meetings are not none then return error
			if len(meetingsCheck) > 0 {
				return nil, errors.New("Overlapping meeting for email " + participant.Email)
			}
		}

		// ensure rsvp chosen values
		if participant.RSVP == "Yes" || participant.RSVP == "No" || participant.RSVP == "Maybe" || participant.RSVP == "Not Answered" {
			continue;
		} else {
			return nil, errors.New("Invalid RSVP")
		}
	}

	// insert the record into collection
	result, err := collection.InsertOne(ctx, meeting)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Could not create")
	}
	return result, nil
}

/*
	function Meeting.getMeeting()
	@purpose:
		get individual meeting
	@params:
		db - mongoDb database instance
	@description:
		get collection and context
		design filter with bson and set Meeting.ID as mongo ObjectID
		query the collection and populate Meeting instance
	@return:
		error - error | nil
*/

func (meeting *Meeting) getMeeting (db *mongo.Database) error{
	collection := db.Collection("meetings")
	ctx := getContext(10)

	// design filter
	filter := bson.M{"_id": meeting.ID}

	// query collections
	err := collection.FindOne(ctx, filter).Decode(&meeting)
	return err
}

/*
	function getMeetings()
	@purpose:
		get meetings in a time range and also paginate
	@params:
		db - mongoDb database instance
		st_time - starting time
		en_time - starting time
		page - page number
		pageSize - size of each page
	@description:
		get collection and context
		design filter with bson and query for times between starttime and endtime
		paginate result
	@return:
		[]bson.M - []Meeting | nil
		error - error | nil
*/

func getMeetings (db *mongo.Database, st_time,en_time time.Time, page,pageSize int) ([]Meeting,error){
	collection := db.Collection("meetings")
	ctx := getContext(10)

	// design filter
	filter := bson.M{"start_time" : bson.M{"$gte" : st_time}, "end_time" : bson.M{"$lte" : en_time}}

	var meetings []Meeting
	// find meetings with filter
	cursor,err := collection.Find(ctx,filter)
	if err != nil {
		fmt.Println(err)
	    return nil,err
	}
	// call cursor to populate all objects found into meetings object
	if err = cursor.All(ctx, &meetings); err != nil {
		fmt.Println(err)
	    return nil,err
	}

	// if paging is to be done
	if page == -1 {
		return meetings,nil
	} else {
		// return slice of meeting data for page
		if len(meetings) >=  pageSize * page {
			return meetings[(page-1) * pageSize : page * pageSize],nil
		} else if len(meetings) <  pageSize * page && len(meetings) > (page-1)* pageSize { 
			return meetings[(page-1) * pageSize : len(meetings)],nil
		} else {
			return nil,nil
		}
	}
}

/*
	function getMeetingsParticipant()
	@purpose:
		get meetings with a participant email and also paginate
	@params:
		db - mongoDb database instance
		email - participant email
		page - page number
		pageSize - size of each page
	@description:
		get collection and context
		design filter with bson and query for email
		paginate result
	@return:
		[]bson.M - []Meeting | nil
		error - error | nil
*/

func getMeetingsParticipant (db *mongo.Database, email string, page,pageSize int) ([]Meeting,error){
	collection := db.Collection("meetings")
	ctx := getContext(10)

	// design filter
	filter := bson.M{"participants.email": email}

	// query collections
	var meetings []Meeting
	cursor,err := collection.Find(ctx,filter)
	if err != nil {
		fmt.Println(err)
	    return nil,err
	}
	if err = cursor.All(ctx, &meetings); err != nil {
	    return nil,err
	}

	// pagination
	if page == -1 {
		return meetings,nil
	} else {
		// return slice of meeting data for page
		if len(meetings) >= pageSize * page {
			return meetings[(page-1)*pageSize : page*pageSize],nil
		} else if len(meetings) < pageSize * page && len(meetings) > (page-1)*pageSize { 
			return meetings[(page-1)*pageSize : len(meetings)],nil
		} else {
			return nil,nil
		}
	}
}