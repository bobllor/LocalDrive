## About

LocalDrive is a self-hosted file storage.

It is built with Go, Python, Bash, React TS, Docker, and MySQL.

## Requirements

The host machine *must be a Unix device*. WSL is supported.

Software requirements:
- Go >=1.25.0
- Node.js >= 22.22.1
- npm >= 10.9.4
- MySQL >= 15.1
- Git
- Docker
- Docker Compose

Optionally, `make` (>= 4.3) is used for task running but can be replaced with Bash instead. 

## Task Running

The project supports running tasks to assist with creation, development, setup, and production. It uses
support scripts and tools found in the `tools` folder.

Task running is *expected to be used with `make`*, and the documentation is written in support of this,
unless stated otherwise.

If preferred, running the shell scripts via Bash can be done for certain tasks found in the `tools` folder.
The file names match their respective use case of the `make` commands.

## Development

The following goes over how to setup the dev environment. *All steps must be followed*
in order to *start working* on a local dev environment.

The steps assume you are in the root project folder.

### Configuration Files

The configuration YAML must be set up for the local dev environment:

```yml
database: 
  name: TestLocalCloudStorage
  address: :3307
  network_protocol: tcp
  file_user:
    username: root
  account_user:
    username: root
storage_path: ./testapp
server_address: :8080
```

The frontend service must also have its own `.env` located in `frontend/.env`:

```env
VITE_SERVER_BASE_URL=http://localhost:8080
```

### Commands

Install the modules and dependencies for Go and `npm`:

```sh
go mod download

make npm-install
```

Create and start the test DB:

```sh
make start-testdb
```

When the command is ran, a new Docker container and volume is created, both named `lcstestdb-l`.

A test file storage is created in the project root folder named `testapp`, and will contain the
files stored for accounts in the `UserAccount` database. The default files created are based
on *the setup script `sql/scripts/00.testdb_setup.sql`*.
- This is used to *hold the stored files* for the storage of the project.

Afterwards, run the commands to start each service, which *requires two separate terminal instances*:

```sh
# service 1: backend server
go run app.go

# service 2: frontend server
make npm-rundev
```