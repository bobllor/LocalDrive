## About

This is a for fun project of mine that mimics a cloud storage. Used primarily
for learning how the all the systems interacts with each other: front end, back end,
network, security, authentication, and more.

It is built with Go, Python, Bash, React TS, Docker, and MariaDB.

## Requirements

The host machine *must be a Unix device*. WSL is supported.

Software requirements:
- Go >=1.25.0
- Node.js >= 22.22.1
- npm >= 10.9.4
- Git
- Docker
- Docker Compose

## Development

The configuration YAML must be set up for the local dev environment.

```yml
database: 
  name: TestLocalCloudStorage
  address: :3307
  network_protocol: tcp
  file_user:
    username: root
  account_user:
    username: root
storage_path: ./app
server_address: :8080
```

The frontend service must also have its own `.env` located in `frontend/.env`:

```env
VITE_SERVER_BASE_URL=http://localhost:8080
```