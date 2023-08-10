## How to run Nuclio

**Step 0:** Make sure you're on branch `1.11.x`.

**Step 1:** Build your local Nuclio version.

```shell
make build
```

**Step 2a:** Run a local Docker registry.

```shell
docker run --rm -d -p 5000:5000 --name registry registry:2
```

**Step 2b:** Add the local registry to the `daemon.json` file or, if you're using Docker Desktop, to `Settings`&rarr;`Docker Engine`. Click [this](https://docs.docker.com/registry/insecure/) for more information.
*(You only have to do this once.)*

```JSON
"insecure-registries": [
    "registry:5000",
    "localhost:5000"
  ]
```

**Step 3:** Run your local Nuclio version.

```shell
ARCHITECTURE=$(uname -m)

if [[ ARCHITECTURE -eq "arm64" ]]; then
  ARCH="arm64"
else
  ARCH="amd64"
fi

COMMAND="docker run \
    --rm -p 8070:8070 \
    -v /var/run/docker.sock:/var/run/docker.sock \
    --name nuclio-dashboard \
    -e NUCLIO_DASHBOARD_REGISTRY_URL='registry:5000' \
    -e NUCLIO_DASHBOARD_RUN_REGISTRY_URL='registry:5000' \
    -e NUCLIO_DASHBOARD_NO_PULL_BASE_IMAGES='true' \
    quay.io/nuclio/dashboard:latest-$ARCH"

eval "$COMMAND"
```

**TODO:** nuctl zum laufen bringen

**TODO:** Ein Knopf um alles zu beenden
- registry stop und weg + volume
- nuclio storage reader stop und weg + volume 
- Die von Nuclio erstellten Container für die Funktionen auch weg  

## ProFaaStinate

- Requests kommen im Dashboad an, im Code: nuclio &rarr; pkg &rarr; dashboard &rarr; resource &rarr; invocation.go &rarr; `handleRequest(...)`
- Go runtime: nuclio &rarr; pkg &rarr; processor &rarr; runtime &rarr; golang &rarr; runtime.go &rarr; ProcessEvent