# File Management with OpenFGA

A simple Go implementation for secure file management using OpenFGA for Relationship-Based Access Control (ReBAC) and MongoDB for storage.

## Available commands

Run all commands from the project root.

### Copy .env file files

```
make config
```

Copy environment config from .env.example to .env
Update configuration as required

### Build the project

```
make build
```

### Run the project

```
make run
```

### Spin up local development container

```
make up
```

Use `docker logs -f openfga` to view openfga logs

### Shut down local development container

```
make down
```
