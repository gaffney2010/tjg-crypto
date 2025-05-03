# tjg-crypto
RPC service to solve cryptograms

The problem that I want to solve is that I have sports data with teams that are written differently in different databases.  For example "mighigan-state" may be "msu" or "MichiganState" or "msu-spartans" in another database.

I've found that the best way to solve this is to compare two seasons and whenever a have a known team pair up with a team that is known in one database but unknown in another, I can make a new mapping.  In this way, the program resembles solving a cryptogram.

APIs will:
- Accept a source-of-truth set of games
- Accept another datasource of games with an initial mapping
- Get source-of-truth team name from a variant.

For the moment, I'm using a placeholder helloworld.

## API

See proto/service.proto.

## Design

We write this in golang as a gRPC service, using PostGRES as the backend database.

We make use of two primary tables:

```
CREATE TABLE IF NOT EXISTS primary (
    namespace VARCHAR(255) NOT NULL,
    date INTEGER NOT NULL,
    team_a VARCHAR(255) NOT NULL,
    team_b VARCHAR(255) NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_namespace_date
ON primary(namespace, date);

CREATE TABLE IF NOT EXISTS mapping (
    namespace VARCHAR(255) NOT NULL,
    secondary VARCHAR(255) NOT NULL,
    primary VARCHAR(255) NOT NULL,
    PRIMARY KEY (namespace, secondary)
);
```

## Test

To test, start up the docker containers, then:

```
grpcurl -plaintext -d '{"namespace": "<TESTNS>", "games": [{"date": 1, "team_a": "A", "team_b": "B"}, {"date": 2, "team_a": "C", "team_b": "B"}, {"date": 2, "team_a": "C", "team_b": "D"}' localhost:50051 crypto.CryptoService/PrimarySeasonReq
```

```
grpcurl -plaintext -d '{"namespace": "<TESTNS>", "games": [{"date": 1, "team_a": "a", "team_b": "b"}, {"date": 2, "team_a": "b", "team_b": "c"}, {"date": 2, "team_a": "d", "team_b": "c"}, "seed": {"secondary": "a", "primary": "A"}' localhost:50051 crypto.CryptoService/SecondarySeasonReq
```

Should return map {"c": "C", "d": "D", "D": "D"}:

```
grpcurl -plaintext -d '{"namespace": "<TESTNS>", "teams": ["c", "d", "D", "e"]}' localhost:50051 crypto.CryptoService/MapTeams
```

Clean up

```
grpcurl -plaintext -d '{"namespace": "<TESTNS>"}' localhost:50051 crypto.CryptoService/DeleteNamespace
```

## Instructions

To run for the first time, run:
```
cd go-service
go mod tidy
```

To update the proto generated files:  Run `protoc --go_out=proto --go-grpc_out=. ../proto/service.proto --proto_path=../proto` from `go-service`.  (May need to install some packages.)


