#!/usr/bin/env bash
echo "run this for db access: psql -U postgres, then select count(*), function_name from delayed_calls group by function_name;"
docker run --name postgres --rm --network profaastinate -p 5432:5432 -e POSTGRES_PASSWORD=1234 -e POSTGRES_DB=nuclio postgres