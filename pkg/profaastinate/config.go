package profaastinate

import (
	"fmt"
	"strconv"
)

const (
	connString        = "postgres://postgres:1234@postgres:5432/postgres"
	insertDelayedCall = `INSERT INTO delayed_calls (
                           function_name, 
                           call_time, 
                           deadline, 
                           HTTP_verb, 
                           headers, 
                           body
                           ) VALUES ($1, $2, $3, $4, $5, $6);`
	callAmountsQuery = `SELECT function_name, COUNT(id) amount FROM delayed_calls GROUP BY function_name;`
	allCallsQuery    = `SELECT * FROM delayed_calls;`
	deleteCallsQuery = `DELETE FROM delayed_calls WHERE id <= $1;`
)

func UrgentCallsQuery(ms, minResults int) string {
	if ms < 0 {
		panic("So nich!")
	}
	sql := `
DELETE FROM delayed_calls WHERE id IN (
	SELECT id FROM delayed_calls
	WHERE deadline <= now() + interval '%s milliseconds'
	UNION
	(SELECT id
	FROM delayed_calls
	ORDER BY deadline ASC
	LIMIT %d)
)
RETURNING *;
`
	sql = fmt.Sprintf(sql, strconv.Itoa(ms), minResults)
	return sql
}

// GenerateListOfIds generates list of ids of function calls to use in DELETE statement
func GenerateListOfIds(calls map[string][]FunctionCall) string {
	s := "("
	for _, fcall := range calls {
		for i, call := range fcall {
			s += strconv.Itoa(call.id)
			if i != len(fcall)-1 {
				s += ", "
			}
		}
	}
	return s + ")"
}
