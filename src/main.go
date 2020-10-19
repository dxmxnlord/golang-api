package main

/*
	function main()

	@purpose:
		run application
	@description:
		create empty Api object and call the Init and Run methods
*/

func main() {
	api := Api{}
	api.Init("appointy-api")
	api.Run(":8080")
}