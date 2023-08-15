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
	"time"
)

const (
	insertDelayedCall = "INSERT INTO delayed_calls (function_name, call_time, HTTP_verb, headers, body) VALUES ($1, $2, $3, $4, $5);"
	connString        = "postgres://postgres:1234@localhost:5432/postgres"
)

type Procrastinator struct {
	conn   *pgx.Conn
	logger logger.Logger
}

func ensureTablesExist(conn *pgx.Conn) error {
	_, err := conn.Exec(context.Background(), "CREATE TABLE IF NOT EXISTS function_metadata(function_name VARCHAR ( 50 ) NOT NULL PRIMARY KEY, max_latency INT NOT NULL);")
	if err != nil {
		return err
	}
	_, err = conn.Exec(context.Background(), "CREATE TABLE IF NOT EXISTS delayed_calls(id SERIAL PRIMARY KEY, function_name VARCHAR ( 50 ) NOT NULL, call_time TIMESTAMP NOT NULL, request TEXT NOT NULL);")
	if err != nil {
		return err
	}
	return nil
}

func NewProcrastinator(logger logger.Logger) Procrastinator {
	// create DB connection
	//conn, err := pgx.Connect(context.Background(), "postgres:1234@postgres:5432") -- old version TODO check which connString works
	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	err = ensureTablesExist(conn)
	return Procrastinator{
		conn, logger,
	}
}

func (pro *Procrastinator) Procrastinate(request *http.Request) error {

	pro.logger.Debug("Procrastinating request for function %s\n", request.Header.Get("x-nuclio-function-name"))

	// read request data
	name := request.Header.Get("x-nuclio-function-name") // TODO does Nuclio already check if the header exists?

	var timestamp time.Time
	locStr := "Europe/Berlin"
	location, locErr := time.LoadLocation(locStr)
	if locErr == nil {
		timestamp = time.Now().In(location)
	} else {
		pro.Logger.Debug("could not resolve location " + locStr)
		timestamp = time.Now()
	}

	httpVerb := request.Method
	headers, headersErr := json.Marshal(request.Header) // string() required
	body, bodyErr := io.ReadAll(request.Body)           // string() required

	// check if all required values exist TODO what happens if not?
	if name == "" || headersErr != nil || bodyErr != nil {
		errMsg := "Error occurred while reading request fields"
		pro.logger.Error(errMsg)
		return errors.New(errMsg)
	}

	// insert values into DB
	// TODO which context to use?
	res, err := pro.conn.Exec(context.Background(), insertDelayedCall, name, timestamp, httpVerb, string(headers), string(body))
	pro.logger.Info(res.String())
	if err != nil {
		pro.logger.Error("Error occurred while inserting request values into database")
	}

	return err
}
