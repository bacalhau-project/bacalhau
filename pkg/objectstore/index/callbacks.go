package index

import (
	"fmt"
)

// UserCallback is a user supplied callback. During an Update or Delete
// action on the database, we want to allow the user to trigger other
// commands to be run against the database.  Typically these Commands
// will be actions adding the object's identifier to other lists or
// hashes in the database. To this end a user can register these functions
// for a specific action against a specific type. e.g. we might register
// a UserCallback for Jobs, which might then generate commands to index
// some of the fields we want to later find it by.
type UserCallback func(any) ([]IndexCommand, error)

type CallbackHooks struct {
	UpdateHooks map[string]UserCallback
	DeleteHooks map[string]UserCallback
}

func NewCallbackHooks() *CallbackHooks {
	return &CallbackHooks{
		UpdateHooks: make(map[string]UserCallback),
		DeleteHooks: make(map[string]UserCallback),
	}
}

func (c *CallbackHooks) RegisterUpdate(prefix string, callback UserCallback) {
	c.UpdateHooks[prefix] = callback
}

func (c *CallbackHooks) RegisterDelete(prefix string, callback UserCallback) {
	c.DeleteHooks[prefix] = callback
}

func (c *CallbackHooks) TriggerUpdate(prefix string, object any) ([]IndexCommand, error) {
	callback, present := c.UpdateHooks[prefix]
	if !present {
		return nil, fmt.Errorf("failed to process update callback hook for %s", prefix)
	}

	return callback(object)
}

func (c *CallbackHooks) TriggerDelete(prefix string, object any) ([]IndexCommand, error) {
	callback, present := c.DeleteHooks[prefix]
	if !present {
		return nil, fmt.Errorf("failed to process delete callback hook for %s", prefix)
	}

	return callback(object)
}
