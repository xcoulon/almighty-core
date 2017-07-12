package model

//WorkItem the WorkItem type of resource to (un)marshall in the JSON-API requests/responses
type WorkItem struct {
	ID      string `jsonapi:"primary,workitems"`
	Version int    `jsonapi:"attr,version"`
}
