package slackbot

import (
	"reflect"

	"github.com/slack-go/slack/slackevents"
)

// Generic filter interface for matching events to a handler
type EventFilter func(evt interface{}) bool

func NewEventTypeEventFilter(example interface{}) EventFilter {
	return func(evt interface{}) bool {
		return reflect.TypeOf(evt) == reflect.TypeOf(example)
	}
}

func NewChannelEventFilter(channelId string) EventFilter {
	return func(evt interface{}) bool {
		switch evt.(type) {
		case slackevents.AppHomeOpenedEvent:
			return evt.(slackevents.AppHomeOpenedEvent).Channel == channelId
		case slackevents.AppMentionEvent:
			return evt.(slackevents.AppMentionEvent).Channel == channelId
		case slackevents.LinkSharedEvent:
			return evt.(slackevents.LinkSharedEvent).Channel == channelId
		case slackevents.MemberJoinedChannelEvent:
			return evt.(slackevents.MemberJoinedChannelEvent).Channel == channelId
		case slackevents.MessageEvent:
			return evt.(slackevents.MessageEvent).Channel == channelId
		case slackevents.PinAddedEvent:
			return evt.(slackevents.PinAddedEvent).Channel == channelId
		case slackevents.PinRemovedEvent:
			return evt.(slackevents.PinRemovedEvent).Channel == channelId

		// other event types don't contain a channel
		default:
			return false
		}
	}
}
