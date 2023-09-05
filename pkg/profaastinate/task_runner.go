package profaastinate

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/nuclio/logger"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// TODO explain
//
//

type FunctionCall struct {
	id        int
	fname     string
	timestamp time.Time
	deadline  time.Time
	verb      string
	headers   map[string][]string
	body      string
}

func NewFunctionCall(id int, fname string, timestamp time.Time, deadline time.Time, verb string, headers map[string][]string, body string) *FunctionCall {
	return &FunctionCall{
		id,
		fname,
		timestamp,
		deadline,
		verb,
		headers,
		body,
	}
}

// NewFunctionCallFromDB uses the results of a query to create a new function call
func NewFunctionCallFromDB(rows *pgx.Rows, logger logger.Logger) *FunctionCall {
	var (
		id        int
		name      string
		timestamp time.Time
		deadline  time.Time
		verb      string
		headers   string
		body      string
	)
	err := (*rows).Scan(&id, &name, &timestamp, &deadline, &verb, &headers, &body)
	if err != nil {
		logger.Debug(err.Error())
	}
	return NewFunctionCall(id, name, timestamp, deadline, verb, deserializeHeaders(headers, logger), body)
}

func (f *FunctionCall) String() string {
	return fmt.Sprintf("call(id=%d, name=%s, deadline=%s)", f.id, f.fname, f.deadline.String())
}

//////////////////////////////
// hustler, fka. taskrunner //
//////////////////////////////

type Taskrunner struct {
	conn   *pgx.Conn
	Logger logger.Logger
}

func NewTaskrunner() *Taskrunner {

	// database connection
	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	err = ensureTablesExist(conn)

	return &Taskrunner{
		conn,
		nil,
	}
}

func (t *Taskrunner) Start(sleepFor time.Duration) {

	stopIt := false
	//go t.boredSupervisor("helloworld", 10, 2, &stopIt)
	go t.swampedSupervisor(&stopIt, 2, 60_000, 60_000)

	time.Sleep(120 * time.Second)
	t.Logger.Info("Exiting bored supervisor")
	stopIt = true

	t.Logger.Info("Finished with bored supervisor")
}

// A swampedSupervisor performs urgent calls for _all_ functions
// urgencyMs determines how close a call's deadline has to be to make it urgent (e.g. 10_000ms)
// frequencyMs
func (t *Taskrunner) swampedSupervisor(stopIt *bool, nWorkers, urgencyMs, frequencyMs int) {

	t.Logger.Debug("swampedSupervisor started")

	// create and start the workers
	tasks := make(chan FunctionCall)
	defer close(tasks)
	t.Logger.Debug("starting workers")
	for i := 0; i < nWorkers; i++ {
		workerId := "swampedWorker" + strconv.Itoa(i)
		t.Logger.Debug("starting worker %s", workerId)
		go t.worker(tasks, workerId)
	}

	// every 'freqencyMs' seconds, look for new urgent calls and send them to the workers
	ticker := time.NewTicker(time.Duration(frequencyMs) * time.Millisecond)
	defer ticker.Stop()
	i := -1
	for range ticker.C {

		i++
		t.Logger.Debug("iteration %d", i)

		// if the taskrunner tells the swampedSupervisor to stop, end the function
		if *stopIt {
			goto theEnd // This is _very_ clean code, @google -- I'll be hearing from you
		}

		// get the calls from the database
		calls := t.getUrgentCalls(urgencyMs)
		nCalls := 0
		for _, f := range calls {
			for range f {
				nCalls++
			}
		}
		t.Logger.Debug("found %d calls", nCalls)
		if len(calls) == 0 {
			t.Logger.Error("no calls for swampedSupervisor")
			continue
		}

		// send calls to workers
		for _, callsForName := range calls {
			for _, call := range callsForName {
				tasks <- call
				t.Logger.Debug("Sent call %s to workers", call.String())
			}
		}
	}

theEnd:
	t.Logger.Debug("SwampedSupervisor is finished")
}

// getUrgentCalls gets and deletes the urgent calls from the database
// urgencyMs: e.g., calls that are due in 10_000 ms are urgent
func (t *Taskrunner) getUrgentCalls(urgencyMs int) map[string][]FunctionCall {

	ctx := context.Background()
	calls := make(map[string][]FunctionCall)

	// begin and defer the commit of a new transaction
	tx, err := t.conn.Begin(context.Background())
	if err != nil {
		return nil
	}
	defer func(tx pgx.Tx, ctx context.Context) {
		err := tx.Commit(ctx)
		if err != nil {

		}
	}(tx, context.Background())

	// get the urgent calls
	rows, err := tx.Query(ctx, UrgentCallsQuery(urgencyMs))
	t.Logger.Debug("Urgent calls query: " + UrgentCallsQuery(urgencyMs))
	if err != nil {
		t.Logger.Error(err)
		t.Logger.Error("could not retrieve urgent calls from delayed_calls: " + err.Error())
	}

	// store the urgent calls in a map
	for rows.Next() {
		call := NewFunctionCallFromDB(&rows, t.Logger)
		calls[call.fname] = append(calls[call.fname], *call)
	}

	// delete calls from DB
	_, err = tx.Exec(context.Background(), `DELETE FROM delayed_calls WHERE id = any($1)`, getIds(calls))
	if err != nil {
		t.Logger.Warn("Error while trying to delete urgent calls from DB")
	}

	// log all urgent ids for debugging
	t.Logger.Debug("Ids of urgent function calls: ")
	for _, fcalls := range calls {
		for _, call := range fcalls {
			t.Logger.Debug(call.id)
		}
	}

	return calls
}

// The boredSupervisor gets 'batchSize' calls for a specific function at a time and performs them
func (t *Taskrunner) boredSupervisor(functionName string, batchSize int, nWorkers int, stopIt *bool) {

	ctx := context.Background()

	// create workers
	calls := make(chan FunctionCall)
	for i := 0; i < nWorkers; i++ {
		go t.worker(calls, functionName+strconv.Itoa(i))
	}

	for !*stopIt {

		// get calls for functionName
		tx, _ := t.conn.Begin(ctx)
		rows, err := tx.Query(ctx, `SELECT * FROM delayed_calls WHERE function_name = $1 ORDER BY deadline ASC LIMIT $2`, functionName, batchSize)
		if err != nil {
			t.Logger.Error("query failed!", err.Error())
		}

		// iterate over all calls
		var ids []int
		for rows.Next() {
			// forward call to workers
			t.Logger.Debug("supervisor: getting new call to send to workers....")
			call := NewFunctionCallFromDB(&rows, t.Logger)
			calls <- *call
			t.Logger.Debug("supervisor: forwarded call to worker node: %s", call.String())

			// store IDs for later deletion
			ids = append(ids, call.id)
		}

		// delete call from DB
		_, err = tx.Exec(ctx, `DELETE FROM delayed_calls WHERE id = any($1)`, ids)
		if err != nil {
			t.Logger.Warn("bored supervisor encountered error while trying to delete calls from DB: %s\", err.Error()")
		}

		// once all executed calls are deleted from the DB, commit the transaction
		//t.Logger.Debug("supervisor: all calls are deleted!")
		err = tx.Commit(ctx)
		if err != nil {
			t.Logger.Error("error during commit of transaction in supervisor", err.Error())
			return
		}
	}
	t.Logger.Debug("supervisor: closing calls channel to stop all workers...")
	// tell the workers to go home
	close(calls)
}

// A worker receives tasks through a channel and executes functions calls until the channel is closed
func (t *Taskrunner) worker(calls <-chan FunctionCall, workerId string) {

	t.Logger.Debug("worker %s started", workerId)

	// take a call from channel, send the request to Nuclio, and log the response
	client := &http.Client{}
	for call := range calls {

		// create the request
		req, err1 := http.NewRequest(call.verb, NuclioURL(), strings.NewReader(call.body))
		for header, values := range call.headers {
			for _, value := range values {
				if strings.ToLower(header) == "x-nuclio-async" {
					// make it synchronous
					req.Header.Set("x-nuclio-async", "false")
				} else if strings.ToLower(header) == "x-nuclio-async-deadline" {
					// don't send the deadline when calling function synchronously
					continue
				} else {
					req.Header.Set(header, value)
				}
			}
		}
		req.Header.Set("executed-by-taskrunner", "true")

		// perform the request
		t.Logger.Info("Executing asynchronous request %d now", call.id)
		res, err2 := client.Do(req)
		if err1 != nil || err2 != nil {
			t.Logger.Error("Error while sending request to Nuclio")
		}

		// log the response
		body, _ := io.ReadAll(res.Body)
		t.Logger.Info("worker %s: received response %s with body %s", workerId, res.Status, string(body))
	}

	t.Logger.Debug("Worker %s is done", workerId)
}

//////////////////////
// helper functions //
//////////////////////

// getIds returns a slice of ints containing the Ids of the function calls
func getIds(calls map[string][]FunctionCall) []int32 {
	var ids []int32
	for _, fcall := range calls {
		for _, call := range fcall {
			ids = append(ids, int32(call.id))
		}
	}
	return ids
}

// deserializeHeaders takes a json String of HTTP headers and returns them as a map
// with headerField: [value1, value2, ...]
func deserializeHeaders(jsonHeaders string, logger logger.Logger) map[string][]string {
	var headers map[string][]string
	err := json.Unmarshal([]byte(jsonHeaders), &headers)
	if err != nil {
		logger.Error(err.Error())
		logger.Error("Could not deserialize headers: %s", jsonHeaders)
	}
	return headers
}

// NuclioURL returns the URL to send requests to Nuclio itself
// TODO test whether this works on other machines
func NuclioURL() string {
	return "http://host.docker.internal:8070/api/function_invocations"
}
