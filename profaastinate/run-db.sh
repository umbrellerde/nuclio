#!/usr/bin/env bash
docker run --name postgres --rm --network profaastinate -p 5432:5432 -e POSTGRES_PASSWORD=1234 -e POSTGRES_DB=nuclio postgres