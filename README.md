# stemcstudio-web

Experimental Web Server for STEMCstudio

## Build and Run

1. Copy artifacts generated by STEMCstudio application to the generated/ subfolder of this project.

2. Build the web.go server.

```bash
go build ./...
```

3. Configure environment variables.

OAuth Apps are defined in https://github.com/settings/developers for the david-geo-holmes account.

```bash
export GITHUB_APPLICATION_CLIENT_ID=b146ae9c79fa94161b98
export GITHUB_APPLICATION_CLIENT_SECRET=...
```

5. Run the server.

```bash
./stemcstudio-web
```

6. Open a Web Browser

The server is available at localhost:8080

