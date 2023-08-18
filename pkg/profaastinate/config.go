package profaastinate

const (
	connString        = "postgres://postgres:1234@postgres:5432/postgres"
	insertDelayedCall = `INSERT INTO delayed_calls (function_name, call_time, HTTP_verb, headers, body) VALUES ($1, $2, $3, $4, $5);`
	callAmountsQuery  = `SELECT function_name, COUNT(id) amount FROM delayed_calls GROUP BY function_name;`
	allCallsQuery     = `SELECT * FROM delayed_calls;`
	deleteCallsquery  = `DELETE FROM delayed_calls WHERE id <= $1;`
)
