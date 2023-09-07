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

/////////////
// hustler //
/////////////

type Hustler struct {
	conn      *pgx.Conn
	Logger    logger.Logger
	Megavisor *Megavisor
}

func NewHustler(megavisor *Megavisor) *Hustler {

	// database connection
	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	err = ensureTablesExist(conn)

	return &Hustler{
		conn,
		nil,
		megavisor,
	}
}

func (h *Hustler) Start() {

	h.Logger.Debug("Hustler started")

	// params for swamped supervisor
	nWorkers, urgencyMs, frequencyMs := 2, 10_000, 10_000
	// params for bored supervisor
	batchSize := 10
	workersForFunctions := map[string]int{
		"helloworld1": 2,
		"helloworld2": 2,
	}

	// begin by starting swamped supervisor
	stopSwamped, stopBored := false, true
	supervisorDone := make(chan bool)
	go h.swampedSupervisor(&stopSwamped, nWorkers, urgencyMs, frequencyMs, &supervisorDone)

	h.Logger.Debug("Hustler started first swamped supervisor")

	for x := range h.Megavisor.modeChannel {

		// TODO wait until they're completely stopped before starting the new one?
		// it could be the case that transactions overlap (as both supervisors use the same DB connection)
		// which, in turn, would result in 'connection busy'

		h.Logger.Debug("Hustler received new mode: %v", x)

		switch x {
		case Bored:
			h.Logger.Debug("Hustler is stopping the swamped supervisor")
			// stop the swamped supervisor
			stopSwamped, stopBored = true, false
			// wait until it's completely done
			<-supervisorDone
			h.Logger.Debug("Hustler thinks the swamped supervisor is done")
			// start bored supervisor
			go h.slightlyLessBoredSupervisor(batchSize, workersForFunctions, &stopBored, &supervisorDone)
			h.Logger.Debug("Hustler started the bored supervisor")
		case Swamped:
			h.Logger.Debug("Hustler is stopping the bored supervisor")
			// stop the bored supervisor
			stopSwamped, stopBored = false, true
			// wait until it's completely done
			<-supervisorDone
			h.Logger.Debug("Hustler thinks the bored supervisor is done")
			// start swamped supervisor
			go h.swampedSupervisor(&stopSwamped, nWorkers, urgencyMs, frequencyMs, &supervisorDone)
			h.Logger.Debug("Hustler started the swamped supervisor")
		}
	}
}

// A swampedSupervisor performs urgent calls for _all_ functions
// urgencyMs determines how close a call's deadline has to be to make it urgent (e.g. 10_000ms)
// frequencyMs
func (h *Hustler) swampedSupervisor(stopIt *bool, nWorkers, urgencyMs, frequencyMs int, supervisorDone *chan bool) {

	h.Logger.Debug("swampedSupervisor started")

	// create and start the workers
	tasks := make(chan FunctionCall)
	defer close(tasks)
	h.Logger.Debug("starting workers")
	for i := 0; i < nWorkers; i++ {
		workerId := "swampedWorker" + strconv.Itoa(i)
		h.Logger.Debug("starting worker %s", workerId)
		go h.worker(tasks, workerId)
	}

	// every 'freqencyMs' seconds, look for new urgent calls and send them to the workers
	ticker := time.NewTicker(time.Duration(frequencyMs) * time.Millisecond)
	defer ticker.Stop()
	i := -1
	for range ticker.C {

		i++
		h.Logger.Debug("iteration %d", i)

		// if the hustler tells the swampedSupervisor to stop, end the function
		if *stopIt {
			goto theEnd // This is _very_ clean code, @google -- I'll be hearing from you
		}

		// get the calls from the database
		calls := h.getUrgentCalls(urgencyMs)
		nCalls := 0
		for _, f := range calls {
			for range f {
				nCalls++
			}
		}
		h.Logger.Debug("found %d calls", nCalls)
		if len(calls) == 0 {
			h.Logger.Error("no calls for swampedSupervisor")
			continue
		}

		// send calls to workers
		for _, callsForName := range calls {
			for _, call := range callsForName {
				tasks <- call
				h.Logger.Debug("Sent call %s to workers", call.String())
			}
		}
	}

theEnd:
	h.Logger.Debug("SwampedSupervisor is finished")
	*supervisorDone <- true
}

// getUrgentCalls gets and deletes the urgent calls from the database
// urgencyMs: e.g., calls that are due in 10_000 ms are urgent
func (h *Hustler) getUrgentCalls(urgencyMs int) map[string][]FunctionCall {

	ctx := context.Background()
	calls := make(map[string][]FunctionCall)

	// begin and defer the commit of a new transaction
	tx, err := h.conn.Begin(context.Background())
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
	h.Logger.Debug("Urgent calls query: " + UrgentCallsQuery(urgencyMs))
	if err != nil {
		h.Logger.Error(err)
		h.Logger.Error("could not retrieve urgent calls from delayed_calls: " + err.Error())
	}

	// store the urgent calls in a map
	for rows.Next() {
		call := NewFunctionCallFromDB(&rows, h.Logger)
		calls[call.fname] = append(calls[call.fname], *call)
	}

	// delete calls from DB
	_, err = tx.Exec(context.Background(), `DELETE FROM delayed_calls WHERE id = any($1)`, getIds(calls))
	if err != nil {
		h.Logger.Warn("Error while trying to delete urgent calls from DB")
	}

	// log all urgent ids for debugging
	h.Logger.Debug("Ids of urgent function calls: ")
	for _, fcalls := range calls {
		for _, call := range fcalls {
			h.Logger.Debug(call.id)
		}
	}

	return calls
}

func (h *Hustler) slightlyLessBoredSupervisor(batchSize int, workersForFunctions map[string]int, stopIt *bool, supervisorDone *chan bool) {

	// reason: there should only be one bored supervisor in order for the DB connection not to be shared
	// MAYBE: use channel from border supervisor to hustler to make sure the connection is ready for the
	// swamped supervisor to be used when switching between them

	ctx := context.Background()

	query := `SELECT * FROM delayed_calls ORDER BY deadline ASC LIMIT $1;`
	h.Logger.Debug("bored query: %s", query)

	callsForFunctions := make(map[string]chan FunctionCall)

	for !*stopIt {

		// start of transaction
		tx, err := h.conn.Begin(ctx)
		if err != nil {
			h.Logger.Warn("Error while trying to begin new transaction: ", err.Error())
		}

		// gets calls from database
		rows, err := tx.Query(ctx, query, batchSize)
		if err != nil {
			h.Logger.Warn("Error while querying DB for new calls: ", err.Error())
		}

		// read calls into map[fname][]calls
		calls := make(map[string][]FunctionCall)
		for rows.Next() {
			call := NewFunctionCallFromDB(&rows, h.Logger)
			calls[call.fname] = append(calls[call.fname], *call)
		}

		// create workers and forward calls to them
		for functionName, functionCalls := range calls {

			// create new workers & channel
			if _, ok := callsForFunctions[functionName]; !ok {
				// new channel
				callsForFunctions[functionName] = make(chan FunctionCall)
				// new workers
				var nWorkers int
				if _, ok := workersForFunctions[functionName]; ok {
					nWorkers = workersForFunctions[functionName]
				} else {
					// TODO default number of workers?
					nWorkers = 1
				}
				for i := 0; i < nWorkers; i++ {
					go h.worker(callsForFunctions[functionName], fmt.Sprintf("Worker(%s, %d)", functionName, i))
				}
			}

			// forward calls to workers
			for _, call := range functionCalls {
				callsForFunctions[functionName] <- call
			}
		}

		// delete call from DB
		ids := getIds(calls)
		_, err = tx.Exec(ctx, `DELETE FROM delayed_calls WHERE id = any($1)`, ids)
		if err != nil {
			h.Logger.Warn("slightlyLessBoredSupervisor encountered error while trying to delete calls from DB: %s", err.Error())
		}

		// commit transactions
		err = tx.Commit(ctx)
		if err != nil {
			h.Logger.Warn("Error while committing transaction: ", err.Error())
		}
	}

	// close channels
	for _, channel := range callsForFunctions {
		close(channel)
	}

	h.Logger.Debug("slightlyLessBoredSupervisor done")
	*supervisorDone <- true
}

/*// The boredSupervisor gets 'batchSize' calls for a specific function at a time and performs them
func (h *Hustler) boredSupervisor(functionName string, batchSize int, nWorkers int, stopIt *bool) {

	ctx := context.Background()

	// create workers
	calls := make(chan FunctionCall)
	for i := 0; i < nWorkers; i++ {
		go h.worker(calls, functionName+strconv.Itoa(i))
	}

	for !*stopIt {

		// get calls for functionName
		tx, _ := h.conn.Begin(ctx)
		rows, err := tx.Query(ctx, `SELECT * FROM delayed_calls WHERE function_name = $1 ORDER BY deadline ASC LIMIT $2`, functionName, batchSize)
		if err != nil {
			h.Logger.Error("query failed!", err.Error())
		}

		// iterate over all calls
		var ids []int
		for rows.Next() {
			// forward call to workers
			h.Logger.Debug("supervisor: getting new call to send to workers....")
			call := NewFunctionCallFromDB(&rows, h.Logger)
			calls <- *call
			h.Logger.Debug("supervisor: forwarded call to worker node: %s", call.String())

			// store IDs for later deletion
			ids = append(ids, call.id)
		}

		// delete call from DB
		_, err = tx.Exec(ctx, `DELETE FROM delayed_calls WHERE id = any($1)`, ids)
		if err != nil {
			h.Logger.Warn("bored supervisor encountered error while trying to delete calls from DB: %s", err.Error())
		}

		// once all executed calls are deleted from the DB, commit the transaction
		//h.Logger.Debug("supervisor: all calls are deleted!")
		err = tx.Commit(ctx)
		if err != nil {
			h.Logger.Error("error during commit of transaction in supervisor", err.Error())
			return
		}
	}
	h.Logger.Debug("supervisor: closing calls channel to stop all workers...")
	// tell the workers to go home
	close(calls)
}*/

// A worker receives tasks through a channel and executes functions calls until the channel is closed
func (h *Hustler) worker(calls <-chan FunctionCall, workerId string) {

	h.Logger.Debug("worker %s started", workerId)

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
		req.Header.Set("executed-by-hustler", "true")

		// perform the request
		h.Logger.Info("Executing asynchronous request %d now", call.id)
		res, err2 := client.Do(req)
		if err1 != nil || err2 != nil {
			h.Logger.Error("Error while sending request to Nuclio")
		}

		// log the response
		body, _ := io.ReadAll(res.Body)
		h.Logger.Info("worker %s: received response %s with body %s", workerId, res.Status, string(body))
	}

	h.Logger.Debug("Worker %s is done", workerId)
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
