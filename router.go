package slackbot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

// A handler function in the middleware chain. When AddHandler() (or any derivative) is called,
// one of these is created and added to the handler stack. The EventHandlerFunc is responsible for
// processing filters.
type EventHandlerFunc struct {
	handlerFunc func(evt interface{}, api *slack.Client, next func(error)) error

	filters []EventFilter
}

type CommandHandlerFunc struct {
	slashCommand string
	handlerFunc  func(cmd slack.SlashCommand, api *slack.Client) (*slack.Msg, error)
}

type EventListenerConfiguration struct {
	VerifyToken string

	SigningSecret string

	HttpPort int

	EventsApiHttpPath string

	SlashCommandsHttpPath string
}

type Router struct {
	api *slack.Client

	handlers []EventHandlerFunc

	commandHandlers []CommandHandlerFunc
}

func NewRouter(api *slack.Client) Router {
	return Router{
		api:      api,
		handlers: []EventHandlerFunc{},
	}
}

func (r *Router) AddHandler(f func(evt interface{}, api *slack.Client, next func(error)) error, filters ...EventFilter) {
	r.handlers = append(r.handlers, EventHandlerFunc{
		handlerFunc: f,
		filters:     filters,
	})
}

func (r *Router) AddCommandHandler(slashCommand string, handlerFunc func(cmd slack.SlashCommand, api *slack.Client) (*slack.Msg, error)) {
	r.commandHandlers = append(r.commandHandlers, CommandHandlerFunc{slashCommand, handlerFunc})
}

func (r *Router) doesEventMatch(hf EventHandlerFunc, evt interface{}) bool {
	matched := true // start with true so that if no filters are specified, all events match
	for _, filter := range hf.filters {
		matched = filter(evt)
		if matched {
			break
		}
	}

	return matched
}

func (r *Router) handleEvent(evt interface{}) error {
	var createHandlerFunc func(idx int, evt interface{}, err error)
	var returnError error

	createHandlerFunc = func(idx int, evt interface{}, err error) {
		next := func(err error) {
			createHandlerFunc(idx+1, evt, err)
		}

		if idx >= len(r.handlers) { // i.e. there is no handler at this index
			returnError = err
			return
		}

		hf := r.handlers[idx]

		// if no filters match, pass to next handler
		if !r.doesEventMatch(hf, evt) {
			next(err)
		}

		// if this event matched our filter match, call this handler
		hf.handlerFunc(evt, r.api, next)
	}

	createHandlerFunc(0, evt, nil)

	return returnError
}

func (r *Router) handleCommand(cmd slack.SlashCommand) (*slack.Msg, error) {
	for _, h := range r.commandHandlers {
		if h.slashCommand == cmd.Command {
			return h.handlerFunc(cmd, r.api)
		}
	}

	return nil, fmt.Errorf("no handler for command %s", cmd.Command)
}

// Begin listening for event callbacks from Slack on the given port and path.
func (r *Router) ListenForEvents(c EventListenerConfiguration) {
	http.HandleFunc(c.EventsApiHttpPath, func(w http.ResponseWriter, req *http.Request) {
		buf := new(bytes.Buffer)
		buf.ReadFrom(req.Body)
		body := buf.String()

		eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body),
			slackevents.OptionVerifyToken(&slackevents.TokenComparator{VerificationToken: c.VerifyToken}))
		if err != nil {
			fmt.Printf("failed to parse event: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
		}

		if eventsAPIEvent.Type == slackevents.URLVerification {
			var r *slackevents.ChallengeResponse
			err := json.Unmarshal([]byte(body), &r)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
			w.Header().Set("Content-Type", "text")
			w.Write([]byte(r.Challenge))
		}

		if eventsAPIEvent.Type == slackevents.CallbackEvent {
			innerEvent := eventsAPIEvent.InnerEvent
			fmt.Printf("info: received event: %v\n", innerEvent)
			err := r.handleEvent(innerEvent.Data)

			if err != nil {
				fmt.Printf("handler for %v failed: %v\n", innerEvent.Data, err)
				w.WriteHeader(http.StatusInternalServerError)
			}
		}
	})

	http.HandleFunc(c.SlashCommandsHttpPath, func(w http.ResponseWriter, req *http.Request) {
		verifier, err := slack.NewSecretsVerifier(req.Header, c.SigningSecret)
		if err != nil {
			fmt.Printf("failed to verify secret: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		req.Body = ioutil.NopCloser(io.TeeReader(req.Body, &verifier))
		cmd, err := slack.SlashCommandParse(req)
		if err != nil {
			fmt.Printf("failed to parse command: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err = verifier.Ensure(); err != nil {
			fmt.Printf("failed to verify secret: %v\n", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		msg, err := r.handleCommand(cmd)
		if err != nil {
			errString := fmt.Sprintf("command handle failure: %v\n", err)
			msg = &slack.Msg{Text: errString}
		}

		b, err := json.Marshal(msg)
		if err != nil {
			errString := fmt.Sprintf("%s command processing failed: %v\n", cmd.Command, err)
			msg = &slack.Msg{Text: errString}
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	})

	fmt.Println("[INFO] Server listening")
	http.ListenAndServe(fmt.Sprintf(":%d", c.HttpPort), nil)
}
