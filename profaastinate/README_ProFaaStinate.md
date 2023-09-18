## How to run Nuclio

**Step 0:** Make sure you're on branch `1.11.x`.

Create a docker network for all containers to live in: 
```shell
docker network create profaastinate
```

Start the postgres database:
```shell
./run-db.sh
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
if [[ $(uname) == "Darwin" ]]; then
    # Check the CPU architecture on macOS
    if [[ $(uname -m) == "arm64" ]]; then
        ARCH="arm64"
    else
        ARCH="unknown"
        echo "Error: Unknown CPU architecture on macOS"
    fi
elif [[ $(uname) == "Linux" ]]; then
    # Set ARCH to "amd64" for Linux
    ARCH="amd64"
else
    # Unknown OS
    ARCH="unknown"
    echo "Error: Unknown operating system"
fi



COMMAND="docker run \
    --rm -p 8070:8070 \
    -v /var/run/docker.sock:/var/run/docker.sock \
    --name nuclio-dashboard \
    -e NUCLIO_DASHBOARD_NO_PULL_BASE_IMAGES='true' \
    -e NUCLIO_DASHBOARD_EXTERNAL_IP_ADDRESSES="host.docker.internal" \
    --network profaastinate \
    --add-host host.docker.internal:host-gateway \
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

**Useful commands:**
```shell
# call a function using curl
curl "localhost:8070/api/function_invocations" -H "x-nuclio-function-name: check" -H "x-nuclio-function-namespace: nuclio" -H "x-nuclio-async: true" -H "x-nuclio-async-deadline: 30000"

# get replicas
curl "localhost:8070/api/functions/test1/replicas" -H "x-nuclio-function-namespace: nuclio"

# get function logs   
curl "localhost:8070/api/functions/test1/logs/nuclio-nuclio-test1?follow=false" -H "x-nuclio-function-namespace: nuclio"
```

