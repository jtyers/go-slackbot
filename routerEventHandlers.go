package slackbot

import (
	"fmt"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

func (r *Router) AddAppMentionEventHandler(f func(evt *slackevents.AppMentionEvent, api *slack.Client, next func(error)) error) {
	// add a type-asserting wrapper around a call to AddHandler()
	ff := func(evt interface{}, api *slack.Client, next func(error)) error {
		if evt2, ok := evt.(*slackevents.AppMentionEvent); ok {
			f(evt2, api, next)
			return nil

		} else {
			return fmt.Errorf("expected AppMentionEvent, but evt was %T: %v", evt, evt)
		}
	}
	r.AddHandler(ff, NewEventTypeEventFilter(new(*slackevents.AppMentionEvent)))
}

func (r *Router) AddMessageEventHandler(f func(evt *slackevents.MessageEvent, api *slack.Client, next func(error)) error) {
	// add a type-asserting wrapper around a call to AddHandler()
	ff := func(evt interface{}, api *slack.Client, next func(error)) error {
		if evt2, ok := evt.(*slackevents.MessageEvent); ok {
			f(evt2, api, next)
			return nil

		} else {
			return fmt.Errorf("expected MessageEvent, but evt was %T: %v", evt, evt)
		}
	}
	r.AddHandler(ff, NewEventTypeEventFilter(new(*slackevents.MessageEvent)))
}
