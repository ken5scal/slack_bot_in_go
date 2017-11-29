package main

import (
	"fmt"
	"net/http"

	"log"
	"os"

	"bytes"
	"github.com/jasonlvhit/gocron"
	"github.com/ken5scal/lp_admin_slack_app/slack"
	lp "github.com/moneyforward/lpmgt"
	"github.com/pkg/errors"
	"sync"
	"time"
)

var port, apiKey, endpointURL, companyID, slackVerificationToken, slackAccessToken, slackChannel string
var durationToAuditInDay = 1
var location = time.UTC

func init() {
	port = os.Getenv("PORT")
	apiKey = os.Getenv("SECRET")
	endpointURL = os.Getenv("END_POINT_URL")
	companyID = os.Getenv("COMPANY_ID")
	loc := os.Getenv("LOCATION")
	slackVerificationToken = os.Getenv("SLACK_VERIFICATION_TOKEN")
	slackAccessToken = os.Getenv("SLACK_ACCESS_TOKEN")
	slackChannel = os.Getenv("SLACK_CHANNEL")

	if port == "" {
		log.Fatal("$PORT must be set")
	} else if apiKey == "" {
		log.Fatal("$SECRET must be set")
	} else if endpointURL == "" {
		log.Fatal("$END_POINT_URL must be set")
	} else if companyID == "" {
		log.Fatal("$COMPANY_ID must be set")
	} else if slackVerificationToken == "" {
		log.Fatal("$SLACK_VERIFICATION_TOKEN must be set")
	} else if slackAccessToken == "" {
		log.Fatal("$SLACK_ACCESS_TOKEN must be set")
	} else if slackChannel == "" {
		slackChannel = "security_audits"
	}

	if loc == "" {
		return
	}

	location = time.FixedZone(loc, 9*60*60)
}

func main() {
	// Slack Client
	slackClient, err := slack.NewClient(slackAccessToken, os.Getenv("DEBUG") != "")
	if err != nil {
		log.Fatal("[ERROR] failed slack client setup: ", err)
	}

	// Schedule and Start Job
	gocron.Every(1).Minutes().Do(auditJob, slackClient)
	_, nextRun := gocron.NextRun()
	fmt.Println(nextRun)
	gocron.Start()

	http.HandleFunc("/", ping)
	http.HandleFunc("/audit", auditHandler)
	http.ListenAndServe(":"+port, nil)
}

func ping(res http.ResponseWriter, req *http.Request) {
	fmt.Fprintln(res, "pong")
}

func auditJob(client *slack.Client) {
	lpClient, err := lp.NewClient(apiKey, endpointURL, companyID, os.Getenv("DEBUG") != "")
	if err != nil {
		log.Printf("[ERROR] Failed lastpass client setup")
	}

	if _, err := client.ChatPostMessage(auditLastPass(lpClient), slackChannel); err != nil {
		log.Printf("[ERROR] Failed post a Message: %s", err)
	}
}

func auditHandler(res http.ResponseWriter, req *http.Request) {
	client, err := lp.NewClient(apiKey, endpointURL, companyID, os.Getenv("DEBUG") != "")
	if err != nil {
		log.Fatal("failed client setup")
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(req.Body)

	// Parse Slash Command Query: https://api.slack.com/slash-commands
	slashCommand, err := slack.BuildSlashCommandRequestFromQuery(buf.String())
	if err != nil {
		log.Printf("[ERROR] Failed to Parse Query from slack: %s", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Validate the command by comparing a token issued by Slack
	if slashCommand.Token != slackVerificationToken {
		log.Printf("[ERROR] Token within slash command %s is not valid", slashCommand.Token)
		res.WriteHeader(http.StatusForbidden)
		return
	}

	// Temporarily returning response.
	res.WriteHeader(http.StatusOK)
	fmt.Fprintln(res, "Got it start Auditing")

	if err := slashCommand.SendResponse(auditLastPass(client)); err != nil {
		log.Printf("[ERROR] Failed to send a audit result: %s", err)
	}
}

func auditLastPass(c *lp.LastPassClient) string {
	folders := make([]lp.SharedFolder, 5)
	events := make([]lp.Event, 5)
	organizationMap := make(map[string][]lp.User)

	c1 := make(chan []lp.User)
	c2 := make(chan []lp.SharedFolder)
	c3 := make(chan *lp.Events)

	// Fetch Data
	numOfGoRoutines := 3 // Change this number based on goroutine to fetch data from LastPass.
	var wg sync.WaitGroup
	wg.Add(numOfGoRoutines)
	go getAllUsers(&wg, lp.NewUserService(c), c1)
	go getSharedFolders(&wg, lp.NewFolderService(c), c2)
	go getEvents(&wg, lp.NewEventService(c), c3, time.Duration(durationToAuditInDay))
	for i := 0; i < numOfGoRoutines; i++ {
		select {
		case users := <-c1:
			for _, u := range users {
				if u.IsAdmin {
					organizationMap["admin"] = append(organizationMap["admin"], u)
				}
				if u.Disabled {
					organizationMap["disabled"] = append(organizationMap["disabled"], u)
				}
				if u.NeverLoggedIn {
					organizationMap["inactive"] = append(organizationMap["inactive"], u)
				}
			}
		case folders = <-c2:
		case es := <-c3:
			events = es.Events
		}
	}
	wg.Wait()

	var out string
	// Pull Admin Users from fetched data. Output string is also constructed
	out = out + fmt.Sprintf("# Admin Users\n")
	for _, u := range organizationMap["admin"] {
		out = out + fmt.Sprintf("- %v\n", u.UserName)
		for _, event := range events {
			if u.UserName == event.Username {
				activity := event.Time.UTC().In(location).String() + " " + event.IPAddress + " " + event.Action + " " + event.Data
				out = out + fmt.Sprintf("	- %v\n", activity)
			}
		}
	}

	// Pull Activities done through LastPassAPI
	out = out + fmt.Sprintf("# API Activities\n")
	for _, event := range events {
		if event.Username == "API" {
			activity := event.Time.UTC().In(location).String() + " " + event.IPAddress + " " + event.Action + " " + event.Data
			out = out + fmt.Sprintf("%v\n", activity)
		}
	}

	// Pull activities to be audited such as re-uses of LastPassword master-password.
	out = out + fmt.Sprintf("\n# Audit Events\n")
	for _, event := range events {
		if event.IsAuditEvent() {
			out = out + fmt.Sprintf("%v\n", event.String(location))
		}
	}

	// Check anyone who can access super-admin credentials on critical infrastructure.
	out = out + fmt.Sprintf("\n# Super-Shared Folders\n")
	for _, folder := range folders {
		if folder.ShareFolderName == "Super-Admins" {
			for _, u := range folder.Users {
				out = out + fmt.Sprintf("- "+u.UserName+"\n")
			}
		}
	}

	// Check disabled users. They may be required to be deleted.
	out = out + fmt.Sprintf("\n# Disabled Users\n")
	for _, u := range organizationMap["disabled"] {
		out = out + fmt.Sprintf("- "+u.UserName+"\n")
	}

	// Check inactive users who never logged in.
	out = out + fmt.Sprintf("\n# Inactive Users\n")
	for _, u := range organizationMap["inactive"] {
		out = out + fmt.Sprintf("- "+u.UserName+"\n")
	}

	return out
}

func getAllUsers(wg *sync.WaitGroup, s *lp.UserService, q chan []lp.User) {
	defer wg.Done()
	users, err := s.GetAllUsers()
	lp.DieIf(errors.Wrap(err, "failed executing GetAllUsers"))
	q <- users
}

func getEvents(wg *sync.WaitGroup, s *lp.EventService, q chan *lp.Events, d time.Duration) {
	defer wg.Done()
	loc, _ := time.LoadLocation(lp.LastPassTimeZone)
	now := time.Now().In(loc)
	dayAgo := now.Add(-d * time.Hour * 24)

	from := lp.JSONLastPassTime{JSONTime: dayAgo}
	to := lp.JSONLastPassTime{JSONTime: now}
	events, err := s.GetAllEventReports(from, to)
	lp.DieIf(errors.Wrap(err, "failed executing GetAllEventReports"))
	q <- events
}

func getSharedFolders(wg *sync.WaitGroup, s *lp.FolderService, q chan []lp.SharedFolder) {
	defer wg.Done()
	folders, err := s.GetSharedFolders()
	lp.DieIf(errors.Wrap(err, "failed executing getSharedFolders"))
	q <- folders
}
