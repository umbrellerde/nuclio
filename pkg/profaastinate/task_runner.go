package profaastinate

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/nuclio/errors"
	"github.com/nuclio/logger"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// TODO port aus main.go nehmen
//var NuclioURL string = "http://localhost" + flag.Lookup("listen-addr").Value.String() + "/api/function_invocations"
// TODO decide which functions should be the taskrunner's methods
// TODO test whether the taskrunner correctly deletes the executed calls

type FunctionCall struct {
	id        int
	fname     string
	timestamp time.Time
	verb      string
	headers   map[string][]string
	body      string
}

func NewFunctionCall(id int, fname string, timestamp time.Time, verb string, headers map[string][]string, body string) *FunctionCall {
	return &FunctionCall{
		id,
		fname,
		timestamp,
		verb,
		headers,
		body,
	}
}

type Taskrunner struct {
	conn   *pgx.Conn
	Logger logger.Logger
}

func NewTaskrunner() *Taskrunner {

	// database connection
	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	err = ensureTablesExist(conn)

	return &Taskrunner{
		conn,
		nil,
	}
}

func (t *Taskrunner) Start(sleepFor time.Duration) {

	// TODO => INIT DB skript

	// TODO for testing -> modify NuclioURL above if this returns the correct port
	port := flag.Lookup("listen-addr").Value.String()
	t.Logger.Error("PORT: %s", port)

	for {

		// sleep
		time.Sleep(sleepFor)
		t.Logger.Debug("taskrunner woke up")

		// get calls
		calls := t.getCalls()
		if len(calls) == 0 {
			t.Logger.Debug("No calls to execute, going back to sleep")
			continue
		}

		// start supervisors
		t.Logger.Debug("taskrunner: starting %d supervisors", len(calls))
		done := make(chan bool)
		for _, functionCalls := range calls {
			go supervisor(functionCalls, 1, done, t.Logger)
		}

		// delete currently executing calls from DB
		maxId, err := getMaxId(calls)
		if err != nil {
			t.Logger.Error(err.Error())
			t.Logger.Error("could not find max id of calls --> were there any to execute?")
		}
		t.deleteCalls(maxId)

		// wait for the supervisors to finish
		for i := 0; i < len(calls); i++ {
			<-done
		}
		t.Logger.Debug("taskrunner: supervisors have finished")
	}
}

func (t *Taskrunner) getCalls() map[string][]FunctionCall {

	calls := make(map[string][]FunctionCall)

	// get all delayed_calls
	rows, err := t.conn.Query(context.Background(), allCallsQuery)
	if err != nil {
		t.Logger.Error(err)
		t.Logger.Error("could not retrieve rows from delayed_calls: " + err.Error())
	}

	// go over delayed_calls and add to map
	for rows.Next() {

		// read values from current row
		var (
			id          int
			fname       string
			callTime    time.Time
			httpVerb    string
			jsonHeaders string
			body        string
		)
		err := rows.Scan(&id, &fname, &callTime, &httpVerb, &jsonHeaders, &body)
		if err != nil {
			t.Logger.Error(err)
			t.Logger.Error("Error occurred while trying to read row values in to local variable")
		}

		// store in map
		currentCall := *NewFunctionCall(id, fname, callTime, httpVerb, deserializeHeaders(jsonHeaders, t.Logger), body)
		if _, hasKey := calls[fname]; hasKey {
			// add to existing slice, function has been called before
			calls[fname] = append(calls[fname], currentCall)
		} else {
			// first call to the function => create new slice
			calls[fname] = []FunctionCall{currentCall}
		}
	}

	return calls
}

func getMaxId(calls map[string][]FunctionCall) (int, error) {

	// find the largest id
	max := -1
	for _, functionCalls := range calls {
		if nextId := functionCalls[len(functionCalls)-1].id; max < nextId {
			max = nextId
		}
	}

	// error case
	var err error
	if max == -1 {
		err = errors.New("Could not find the max id -- is the calls map empty?")
	}

	return max, err
}

func (t *Taskrunner) deleteCalls(maxId int) {
	t.Logger.Debug("Taskrunner is deleting calls with id < %d", maxId)
	_, err := t.conn.Exec(context.Background(), deleteCallsquery, maxId)
	if err != nil {
		t.Logger.Error(err.Error())
		t.Logger.Error("error occurred while trying to delete call ids")

	}
}

// there is one supervisor per function
// each supervisor has multiple workers
// the supervisor sends calls to its workers through a channel
func supervisor(calls []FunctionCall, nWorkers int, supervisorDone chan<- bool, logger logger.Logger) {

	if len(calls) == 0 {
		logger.Error("no calls for supervisor")
		return
	}

	fname := calls[0].fname
	logger.Debug("Supervisor for function %s started", fname)
	tasks := make(chan FunctionCall)
	workerDone := make(chan bool)

	// create workers
	for i := 0; i < nWorkers; i++ {
		workerId := fname + strconv.Itoa(i)
		go worker(tasks, workerDone, logger, workerId)
	}
	// put calls into tasks channel
	for _, call := range calls {
		tasks <- call
	}
	close(tasks)
	// wait for all workers to finish
	for i := 0; i < nWorkers; i++ {
		<-workerDone
	}
	logger.Debug("supervisor of function %s workerDone", calls[0].fname) // asserts that len(calls) > 0 TODO improve

	supervisorDone <- true
}

func worker(calls <-chan FunctionCall, workerDone chan<- bool, logger logger.Logger, workerId string) {

	logger.Debug("worker %s started", workerId)

	client := &http.Client{}
	for call := range calls {
		// send http request to Nuclio and log the response
		// create the request
		req, err1 := http.NewRequest(call.verb, NuclioURL(), strings.NewReader(call.body))
		// set the headers
		for header, values := range call.headers {
			for _, value := range values {
				logger.Debug("%s: (header, value) = (%s, %s)", workerId, header, value)
				if strings.ToLower(header) == "x-nuclio-async" {
					req.Header.Set("x-nuclio-async", "false")
				} else {
					req.Header.Set(header, value) // TODO double check this line
				}
			}
		}
		req.Header.Set("executed-by-taskrunner", "true")
		// perform the request
		logger.Info("Executing asynchronous request %d now", call.id)
		res, err2 := client.Do(req)
		if err1 != nil || err2 != nil {
			logger.Error("Error while sending request to Nuclio")
		}
		// log the response
		body, _ := io.ReadAll(res.Body)
		logger.Info("worker %s: received response %s with body %s", workerId, res.Status, string(body))
	}

	// tell the supervisor that this worker is done
	workerDone <- true

	logger.Debug("Worker %s is done", workerId)
}

// method instead of function to access the taskrunner's logger
func deserializeHeaders(jsonHeaders string, logger logger.Logger) map[string][]string {
	var headers map[string][]string
	err := json.Unmarshal([]byte(jsonHeaders), &headers)
	if err != nil {
		logger.Error(err.Error())
		logger.Error("Could not deserialize headers: %s", jsonHeaders)
	}
	return headers
}

func NuclioURL() string {
	//s := "http://localhost" + flag.Lookup("listen-addr").Value.String() + "/api/function_invocations"
	s := "http://host.docker.internal:8070/api/function_invocations" // TODO what happens if I use localhost:flag.Lookup... here?
	return s
}
