# UserAPI Service

This is a microservice designed to manage access to users, implemented in Go. The service provides both HTTP and gRPC APIs for interacting with user data stored in a MongoDB database.

## Features

- **Add a new User**
- **Modify an existing User**
- **Remove a User**
- **Return a paginated list of Users with filtering capabilities**
- **Notify other services of changes to User Entities**
- **Health checks**

## Getting Started

### Prerequisites

- Go (version 1.18 or later)
- MongoDB
- Docker (for running MongoDB locally)

### Installation

1. Clone the repository:
(Or extract the .zip file given) - The repo below is private
```sh
git clone https://github.com/HaydnG/faceit-task
cd userapi
```

2. Install dependencies:

```sh
go mod tidy
```

3. Ensure MongoDB is running. If you have Docker installed, you can run MongoDB using:

```sh
docker-compose up --build
```

### Running the Service

1. Start the HTTP and gRPC servers:

```sh
go run main.go -httpport=8080 -grpcport=9090
```

### Re-generating from user.proto

```sh
protoc --go_out=. --go-grpc_out=. pb/user.proto
```

This will generate files into the pb directory

### Unit tests / Benchmarks
```sh
go test ./... --cover -count=1
?       userapi/db      [no test files]
?       userapi/health  [no test files]
?       userapi/mocks   [no test files]
?       userapi/pb      [no test files]
ok      userapi 0.472s  coverage: 74.4% of statements
ok      userapi/cacheStore      2.191s  coverage: 83.3% of statements
ok      userapi/data    0.222s  coverage: [no statements]
ok      userapi/validation      0.190s  coverage: 100.0% of statements
```

## Benchmarhs

```sh
pkg: userapi/validation
cpu: AMD Ryzen 7 5800X3D 8-Core Processor
BenchmarkNumber-16      46102778                25.43 ns/op            0 B/op          0 allocs/op
BenchmarkUser-16         3496152               342.2 ns/op             0 B/op          0 allocs/op
PASS
ok      userapi/validation      2.900s
```

## Handler performance
TODO Measure http/gRPC handler performance


### HTTP Endpoints

- **GET /userapi/getall**: Fetches all users. (20 sec cache)
- **GET /userapi/get**: Finds users with a given query.
  - Query parameters: `country`, `nickname`, `createdAfter`, `page`, `limit`.
- **POST /userapi/add**: Creates a new user.
- **POST /userapi/update**: Updates an existing user.
- **POST /userapi/delete**: Deletes a user by ID.
- **GET /userapi/deleteall**: Deletes all users.
- **GET /healthz**: Health check endpoint for both HTTP and gRPC servers.

#### Example HTTP Usage with `curl`

##### 1. **Call AddUser Endpoint**:
- Expected response **(STATUS_OK 200)**
- Failure response **(STATUS_InternalServerError 500)**
```sh
curl 'http://localhost:8080/userapi/add' \
-H 'Content-Type: application/json' \
--data-raw '{
    "first_name": "Razzil",
    "last_name": "Darkbrew",
    "nickname": "Alchemist",
    "password": "moneyMoneyM0n3y",
    "email": "Razzil.Darkbrew@example.com",
    "country": "UK"
}'
```
<details><summary>Example AddUser Response</summary>

```json
{
    "id": "0d0f9944-d902-4db1-b83b-6b25a61f89e2",
    "first_name": "Razzil",
    "last_name": "Darkbrew",
    "nickname": "Alchemist",
    "password": "moneyMoneyM0n3y",
    "email": "Razzil.Darkbrew@example.com",
    "country": "UK",
    "created_at": "2024-06-16T17:32:28.2136171Z",
    "updated_at": "2024-06-16T17:32:28.2136171Z"
}
```
</details>

##### 2. **Call UpdateUser Endpoint**:
- Expected response **(STATUS_OK 200)**
- Failure response **(STATUS_InternalServerError 500)**
```sh
curl 'http://localhost:8080/userapi/update' \
-H 'Content-Type: application/json' \
--data-raw '{
    "ID": "$(id from addUser)",
    "first_name": "Razzil",
    "last_name": "Darkbrew",
    "nickname": "Alchemist",
    "password": "Blink!moneyMoneyM0n3y",
    "email": "Razzil.Darkbrew@example.com",
    "country": "UK"
}'
```
<details><summary>Example UpdateUser Response</summary>

```json
{
    "id": "0d0f9944-d902-4db1-b83b-6b25a61f89e2",
    "first_name": "Razzil",
    "last_name": "Darkbrew",
    "nickname": "Alchemist",
    "password": "Blink!moneyMoneyM0n3y",
    "email": "Razzil.Darkbrew@example.com",
    "country": "UK",
    "created_at": "2024-06-16T17:32:28.213Z",
    "updated_at": "2024-06-16T17:43:38.985Z"
}
```
</details>

##### 3. **Call GetAllUsers Endpoint (20 sec cache)**:
- Expected response **(STATUS_OK 200)**
- Failure response **(STATUS_InternalServerError 500)**
```sh
curl 'http://localhost:8080/userapi/getall'
```
<details><summary>Example UpdateUser Response</summary>

```json
[
    {
        "id": "0d0f9944-d902-4db1-b83b-6b25a61f89e2",
        "first_name": "Razzil",
        "last_name": "Darkbrew",
        "nickname": "Alchemist",
        "password": "Blink!moneyMoneyM0n3y",
        "email": "Razzil.Darkbrew@example.com",
        "country": "UK",
        "created_at": "2024-06-16T17:32:28.213Z",
        "updated_at": "2024-06-16T17:43:38.985Z"
    },
    {
        "id": "a5557cd5-3083-4ecb-a888-71d98ee1e39e",
        "first_name": "Visage",
        "last_name": "joe",
        "nickname": "aXE",
        "password": "VERYSEcure3343",
        "email": "joe.jim@example.com",
        "country": "UK",
        "created_at": "2024-06-16T17:46:01.377Z",
        "updated_at": "2024-06-16T17:46:01.377Z"
    }
]
```
</details>


##### 4. **Call GetUsers Endpoint (Filtered)**:
- Expected response **(STATUS_OK 200)**
- Failure response **(STATUS_InternalServerError 500)**

```sh
curl 'http://localhost:8080/userapi/get?country=UK&nickname=al&createdAfter=2024-06-14T18%3A37%3A47.572Z&page=1&limit=50'
```
<details><summary>Example GetUsers Response</summary>

```json
[
    {
        "id": "0d0f9944-d902-4db1-b83b-6b25a61f89e2",
        "first_name": "Razzil",
        "last_name": "Darkbrew",
        "nickname": "Alchemist",
        "password": "Blink!moneyMoneyM0n3y",
        "email": "Razzil.Darkbrew@example.com",
        "country": "UK",
        "created_at": "2024-06-16T17:32:28.213Z",
        "updated_at": "2024-06-16T17:43:38.985Z"
    }
]
```
</details>

##### 5. **Call Delete User Endpoint**:

- Expected response **(STATUS_OK 200)**
- Failure response **(STATUS_InternalServerError 500)**

```sh
curl --location 'http://localhost:8080/userapi/delete' \
-H 'Content-Type: application/json' \
--data-raw '{
   "ID": "d174809b-5559-4855-a74b-ddf24e993e39"
}'
```

##### 5. **Call Delete All Users Endpoint**:

- Expected response **(STATUS_OK 200)**
- Failure response **(STATUS_InternalServerError 500)**

```sh
curl --location 'http://localhost:8080/userapi/deleteall' -H 'Content-Type: application/json'
```

### gRPC Endpoints

- **UserService.GetAllUsers**: Fetches all users. (20 sec cache)
- **UserService.GetUsers**: Finds users with a given query.
- **UserService.AddUser**: Creates a new user.
- **UserService.UpdateUser**: Updates an existing user.
- **UserService.DeleteUser**: Deletes a user by ID.

```protobuf
user.UserService is a service:
service UserService {
  rpc AddUser ( .user.AddUserRequest ) returns ( .user.User );
  rpc DeleteUser ( .user.DeleteUserRequest ) returns ( .user.Empty );
  rpc GetAllUsers ( .google.protobuf.Empty ) returns ( .user.GetUsersResponse );
  rpc GetUsers ( .user.GetUsersRequest ) returns ( .user.GetUsersResponse );
  rpc UpdateUser ( .user.UpdateUserRequest ) returns ( .user.User );
}
```

#### Example gRPC Usage with `grpcurl`

You can use `grpcurl` to interact with the gRPC service. Here are some examples:

##### User Watcher
To listen for any user changes, you can use the following:
```sh
grpcurl -plaintext -d '{}' localhost:9090 user.UserService/WatchUsers
```
<details><summary>Example Watcher Output (Add + Deletion)</summary>

```json
{
  "userId": "fabf2700-3711-45aa-a4c1-aa479b8ec95d",
  "updateType": "CREATED",
  "user": {
    "ID": "fabf2700-3711-45aa-a4c1-aa479b8ec95d",
    "firstName": "Visage",
    "lastName": "joe",
    "nickname": "aXE",
    "password": "VERYSEcure3343",
    "email": "joe.jim@example.com",
    "country": "UK",
    "createdAt": "2024-06-18T19:34:18.404692100Z",
    "updatedAt": "2024-06-18T19:34:18.404692100Z"
  }
}
{
  "userId": "fabf2700-3711-45aa-a4c1-aa479b8ec95d",
  "updateType": "DELETED",
  "user": {
    "ID": "fabf2700-3711-45aa-a4c1-aa479b8ec95d"
  }
}
```
</details>

##### 1. **Call AddUser Method**:
```sh
grpcurl -plaintext -d '{
    "first_name": "Razzil",
    "last_name": "Darkbrew",
    "nickname": "Alchemist",
    "password": "moneyMoneyM0n3y",
    "email": "Razzil.Darkbrew@example.com",
    "country": "UK"
}' localhost:9090 user.UserService/AddUser
```
<details><summary>Example AddUser Response</summary>

```json
{
  "ID": "ab64160e-5093-442e-b131-fd6ea3d3b17d",
  "firstName": "Razzil",
  "lastName": "Darkbrew",
  "nickname": "Alchemist",
  "password": "moneyMoneyM0n3y",
  "email": "Razzil.Darkbrew@example.com",
  "country": "UK",
  "createdAt": "2024-06-16T17:19:01.140270700Z",
  "updatedAt": "2024-06-16T17:19:01.140270700Z"
}
```
</details>

##### 2. **Call UpdateUser Method**:
```sh
grpcurl -plaintext -d '{
    "ID": "$(id from addUser)",
    "first_name": "Razzil",
    "last_name": "Darkbrew",
    "nickname": "Alchemist",
    "password": "Blink!moneyMoneyM0n3y",
    "email": "Razzil.Darkbrew@example.com",
    "country": "UK"
}' localhost:9090 user.UserService/UpdateUser
```
<details><summary>Example UpdateUser Response</summary>

```json
{
  "ID": "ab64160e-5093-442e-b131-fd6ea3d3b17d",
  "firstName": "Razzil",
  "lastName": "Darkbrew",
  "nickname": "Alchemist",
  "password": "Blink!moneyMoneyM0n3y",
  "email": "Razzil.Darkbrew@example.com",
  "country": "UK",
  "createdAt": "2024-06-16T17:19:01.140Z",
  "updatedAt": "2024-06-16T17:21:49.086Z"
}
```
</details>

##### 3. **Call GetAllUsers Method (20 sec cache)**:
```sh
grpcurl -plaintext localhost:9090 user.UserService/GetAllUsers
```

<details><summary>Example GetAllUsers Response </summary>

```json
{
  "users": [
    {
      "ID": "8c358ed6-adbf-4756-ae97-0f68c8f16876",
      "firstName": "Visage",
      "lastName": "joe",
      "nickname": "aXE",
      "password": "VERYSEcure3343",
      "email": "joe.jim@example.com",
      "country": "UK",
      "createdAt": "2024-06-16T17:08:10.299Z",
      "updatedAt": "2024-06-16T17:08:10.299Z"
    },
    {
      "ID": "ab64160e-5093-442e-b131-fd6ea3d3b17d",
      "firstName": "Razzil",
      "lastName": "Darkbrew",
      "nickname": "Alchemist",
      "password": "Blink!moneyMoneyM0n3y",
      "email": "Razzil.Darkbrew@example.com",
      "country": "UK",
      "createdAt": "2024-06-16T17:19:01.140Z",
      "updatedAt": "2024-06-16T17:21:49.086Z"
    }
  ]
}
```
</details>

##### 3. **Call GetUsers Method (Filtered)**:
```sh
grpcurl -plaintext -d '{
        "country": "UK",
        "nickname": "",
        "created_after": "2024-06-15T17:19:01.140Z",
        "page": "1",
        "limit": "50"
}' localhost:9090 user.UserService/GetUsers
```

<details><summary>Example GetAllUsers Response</summary>

```json
{
  "users": [
    {
      "ID": "8c358ed6-adbf-4756-ae97-0f68c8f16876",
      "firstName": "Visage",
      "lastName": "joe",
      "nickname": "aXE",
      "password": "VERYSEcure3343",
      "email": "joe.jim@example.com",
      "country": "UK",
      "createdAt": "2024-06-16T17:08:10.299Z",
      "updatedAt": "2024-06-16T17:08:10.299Z"
    },
    {
      "ID": "ab64160e-5093-442e-b131-fd6ea3d3b17d",
      "firstName": "Razzil",
      "lastName": "Darkbrew",
      "nickname": "Alchemist",
      "password": "Blink!moneyMoneyM0n3y",
      "email": "Razzil.Darkbrew@example.com",
      "country": "UK",
      "createdAt": "2024-06-16T17:19:01.140Z",
      "updatedAt": "2024-06-16T17:21:49.086Z"
    }
  ]
}
```
</details>

##### 3. **Call DeleteUser Method**:
```sh
grpcurl -plaintext -d '{"ID": "8c358ed6-adbf-4756-ae97-0f68c8f16876"}' localhost:9090 user.UserService/DeleteUser
```

## Code Overview

### main.go

The main file sets up and starts the HTTP and gRPC servers, and handles graceful shutdown.

### HTTP Handlers

- `getAllUsersHandler`: Fetches all users from the database.
- `getUserHandler`: Finds users based on query parameters.
- `addUserHandler`: Adds a new user to the database.
- `updateUserHandler`: Updates an existing user in the database.
- `deleteUserHandler`: Deletes a user by ID.
- `deleteAllUsersHandler`: Deletes all users from the database.

### gRPC Handlers

- `ServiceServer.GetAllUsers`: Fetches all users from the database.
- `ServiceServer.GetUsers`: Finds users based on query parameters.
- `ServiceServer.AddUser`: Adds a new user to the database.
- `ServiceServer.UpdateUser`: Updates an existing user in the database.
- `ServiceServer.DeleteUser`: Deletes a user by ID.

### Health Checks

The health check handler ensures that both HTTP and gRPC servers are ready to serve requests.