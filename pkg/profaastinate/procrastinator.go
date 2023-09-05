package profaastinate

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/nuclio/errors"
	"github.com/nuclio/logger"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Procrastinator struct {
	conn   *pgx.Conn
	Logger logger.Logger
}

func ensureTablesExist(conn *pgx.Conn) error {
	_, err := conn.Exec(context.Background(), `
			CREATE TABLE IF NOT EXISTS delayed_calls(
		    	id SERIAL PRIMARY KEY,
				function_name VARCHAR ( 50 ) NOT NULL,
				call_time TIMESTAMP NOT NULL,
			    deadline TIMESTAMP NOT NULL,
				HTTP_verb TEXT NOT NULL,
				headers TEXT NOT NULL,
				body TEXT NOT NULL
			);
		`)
	return err
}

func NewProcrastinator() *Procrastinator {
	// create DB connection
	conn, err := pgx.Connect(context.Background(), connString)
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

	pro.Logger.Debug("Procrastinating request for function %s\n", request.Header.Get("x-nuclio-function-name"))

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
	deadlineStr := request.Header.Get("x-nuclio-async-deadline")
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
	res, err := pro.conn.Exec(context.Background(), insertDelayedCall, name, timestamp, deadline, httpVerb, string(headers), string(body))
	pro.Logger.Info("Response from database: %s", res.String())
	if err != nil {
		pro.Logger.Debug(err.Error())
		pro.Logger.Error("Error occurred while inserting request values into database")
	}

	return err
}
