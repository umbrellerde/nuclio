## How to run Nuclio

**Step 0:** Make sure you're on branch `1.11.x`.

Create a docker network for all containers to live in: 
```shell
docker network create profaastinate
```

**Step 1:** Build your local Nuclio version.

```shell
cd ..
make build
```

Or 

```shell
make dashboard
```

**Step 2:** Run your local Nuclio version.

```shell
if [[ $(uname -m) -eq "arm64" ]]; then
  ARCH="arm64"
else
  ARCH="amd64"
fi

COMMAND="docker run \
    --rm -p 8070:8070 \
    -v /var/run/docker.sock:/var/run/docker.sock \
    --name nuclio-dashboard \
    -e NUCLIO_DASHBOARD_NO_PULL_BASE_IMAGES='true' \
    --network profaastinate \
    quay.io/nuclio/dashboard:latest-$ARCH"

eval "$COMMAND"
```


## Nuctl

- `--platform local` an `nuctl ...` ranhängen oder `export NUCTL_PLATFORM="local"`







**TODO:** Einen Knopf um alles zu beenden
- registry stop und weg + volume
- nuclio storage reader stop und weg + volume 
- Die von Nuclio erstellten Container für die Funktionen auch weg  

## ProFaaStinate

- Requests kommen im Dashboad an, im Code: nuclio &rarr; pkg &rarr; dashboard &rarr; resource &rarr; invocation.go &rarr; `handleRequest(...)`

**Useful commands:**
```shell
# call a function using curl
curl "localhost:8070/api/function_invocations" -H "x-nuclio-function-name: test1" -H "x-nuclio-function-namespace: nuclio" -H "x-nuclio-async: true"

# get replicas
curl "localhost:8070/api/functions/test1/replicas" -H "x-nuclio-function-namespace: nuclio"

# get function logs   
curl "localhost:8070/api/functions/test1/logs/nuclio-nuclio-test1?follow=false" -H "x-nuclio-function-namespace: nuclio"
```

