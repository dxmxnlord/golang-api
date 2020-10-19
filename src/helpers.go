package main;

import (
	"encoding/json"
	"net/http"
	"context"
	"time"
)

/*
	function getContext()
	@params:
		secs - timeout seconds
	@description:
		return context with a timeout for mongoDb actions
*/

func getContext(secs time.Duration) context.Context {
	ctx, _ := context.WithTimeout(context.Background(), secs*time.Second)
	return ctx
}

/*
	function jsonResponse()
	@params:
		http.ResponseWriter - response object
		status - http code
		data - return data
	@description:
		Marshal data into json format and Write response after setting headers
*/

func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	response, _ := json.Marshal(data)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(response)
}

/*
	function errorResponse()
	@params:
		http.ResponseWriter - response object
		status - http code
		msg - error message
	@description:
		call jsonResponse() with error message as data
*/

func errorResponse(w http.ResponseWriter, status int, msg string) {
	jsonResponse(w, status, map[string]string{"error": msg})
}

