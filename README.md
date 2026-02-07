# Secure File Access Management with OpenFGA

A simple Go implementation for secure file access management using OpenFGA for Relationship-Based Access Control (ReBAC) and MongoDB for storage.

## Authorization Model

The service uses the following relationship logic:

- A user who uploads a file is assigned the owner relation.
- The viewer permission is automatically granted to the owner.
- Only owners can share (or revoke access to) a file with a collaborator.

### Setup the Model

Run this curl command to configure your OpenFGA Store:

```
curl -X POST http://localhost:8080/stores/YOUR_STORE_ID/authorization-models \
  -H "Content-Type: application/json" \
  -d '{
  "schema_version": "1.1",
  "type_definitions": [
    { "type": "user" },
    {
      "type": "file",
      "relations": {
        "owner": { "this": {} },
        "collaborator": { "this": {} },
        "viewer": {
          "union": {
            "child": [
              { "computedUserset": { "relation": "owner" } },
              { "computedUserset": { "relation": "collaborator" } }
            ]
          }
        }
      },
      "metadata": {
        "relations": {
          "owner": { "directly_related_user_types": [{ "type": "user" }] },
          "collaborator": { "directly_related_user_types": [{ "type": "user" }] }
        }
      }
    }
  ]
}'
```

## Running the App

- Clone Repo: `git clone https://github.com/manaraph/go-openfga-implementation.git`
- Navigate to folder: `cd go-openfga-implementation`
- Install dependencies: `go mod tidy`
- Copy configuration to .env: `make config` and update the details with your own configuration.
- Spin up local development container: `make up`
- Run app: `make run`

Note: Ensure the local development container is running and you have mongo DB installed and running.

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
