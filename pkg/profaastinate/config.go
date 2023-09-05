package profaastinate

import "strconv"

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

func UrgentCallsQuery(ms int) string {
	if ms < 0 {
		panic("So nicht!")
	}
	return "SELECT * from delayed_calls WHERE deadline <= now() + interval '" + strconv.Itoa(ms) + " milliseconds';"
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
