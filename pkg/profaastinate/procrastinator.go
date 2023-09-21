package profaastinate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nuclio/errors"
	"github.com/nuclio/logger"
	"github.com/nuclio/nuclio/pkg/common/headers"
)

type Procrastinator struct {
	conn   *pgxpool.Pool
	Logger logger.Logger
}

func ensureTablesExist(conn *pgxpool.Pool) error {
	_, err := conn.Exec(context.Background(), `
			CREATE TABLE IF NOT EXISTS delayed_calls(
		    	id SERIAL PRIMARY KEY,
				function_name VARCHAR ( 50 ) NOT NULL,
				call_time TIMESTAMPTZ NOT NULL,
			    deadline TIMESTAMPTZ NOT NULL,
				HTTP_verb TEXT NOT NULL,
				headers TEXT NOT NULL,
				body TEXT NOT NULL
			);
			ALTER DATABASE postgres SET DEFAULT_TRANSACTION_ISOLATION TO 'serializable';
		`)
	return err
}

func NewProcrastinator() *Procrastinator {
	// create DB connection
	conn, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	err = ensureTablesExist(conn)
	return &Procrastinator{
		conn, nil,
	}
}

func (pro *Procrastinator) Procrastinate(request *http.Request) error {

	pro.Logger.Debug("Procrastinating request for function %s with callId header %s\n", request.Header.Get("x-nuclio-function-name"), request.Header.Get("callid"))

	// read request data
	name := request.Header.Get("x-nuclio-function-name") // TODO does Nuclio already check if the header exists?

	// get current time
	var timestamp time.Time
	locStr := "Europe/Berlin"
	location, locErr := time.LoadLocation(locStr)
	if locErr == nil {
		timestamp = time.Now().In(location)
	} else {
		pro.Logger.Debug("could not resolve location " + locStr)
		timestamp = time.Now()
	}

	// set function deadline
	deadlineStr := request.Header.Get(headers.AsyncCallDeadline)
	deadlineInt, deadlineErr := strconv.Atoi(deadlineStr)
	pro.Logger.Info("Function has deadline of %d ms", deadlineInt)
	if deadlineErr != nil {
		deadlineInt = 0
		pro.Logger.Debug("No function deadline found, using default")
	}
	deadline := timestamp.Add(time.Duration(deadlineInt) * time.Millisecond)

	// get request header
	httpVerb := request.Method
	headers, headersErr := json.Marshal(request.Header)
	body, bodyErr := io.ReadAll(request.Body)

	// check if all required values exist
	if name == "" || headersErr != nil || bodyErr != nil {
		errMsg := "Error occurred while reading request fields"
		pro.Logger.Error(errMsg)
		return errors.New(errMsg)
	}

	// insert values into DB
	res, err := pro.conn.Exec(context.Background(), insertDelayedCall, name, timestamp.Format(time.RFC3339Nano), deadline.Format(time.RFC3339Nano), httpVerb, string(headers), string(body))
	pro.Logger.Info("Response from database: %s", res.String())
	if err != nil {
		pro.Logger.Debug(err.Error())
		pro.Logger.Error("Error occurred while inserting request values into database")
	}

	return err
}
