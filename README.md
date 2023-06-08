# Ivory 🐘

[![Go
Reference](https://pkg.go.dev/badge/github.com/tristanfisher/ivory.svg)](https://pkg.go.dev/github.com/tristanfisher/ivory)
[![Go Report
Card](https://goreportcard.com/badge/github.com/tristanfisher/ivory)](https://goreportcard.com/report/github.com/tristanfisher/ivory)

## Overview

Ivory makes it easy for you to create and manage PostgreSQL databases via a Go program.

This is particularly useful during tests or any other time that you only want a pg database with a lifecycle that is
managed programmatically.

## Usage

When not told to create a database or run SQL, ivory will simply try to connect to a PostgreSQL database, return
database handles, and an function for dropping an existing database and closing DB handles.

However, the primary goal of this library is to make it easy to bootstrap and cleanup databases.

The following is likely all you need to know in order to make use of this library:

- If a database name is provided in the options, a random database name is not generated. Otherwise, a
collision-resistant database name is generated.
- If told to create a database, Ivory will oblige and bind 2 connections:
  1. Server-scoped (connection without database in connection string -- required to drop a database instance)
  2. Database-scoped (connection with database in connection string)
- A slice of strings may be provided for Ivory to be treated as migrations. These may include any valid SQL, including
  transactions.
- A "tear down function" that will drop the created (or specified) database and close database handles is returned.
  Calling this is optional.  Ivory does not implicitly drop databases or close connections.

As an example:

```go
dbOptions := &ivory.DatabaseOptions{
  Host:     "localhost",
  Port:     5555,
  SslMode:  "disable",
  User:     "postgres",
  Password: "rootUserSeriousPassword1",
}
// we discard two db handles here, one to the instance, one scoped to the database in the instance teardown() closes
// database handles and cleans up the created database
_, _, dbName, teardown, err := ivory.New(
  context.TODO(),
  dbOptions,
  []string{"CREATE TABLE grocery_list (item char(25))"},
  true,
  "my_app")
defer teardown()
if err != nil {
  fmt.Println("error creating: ", err)
  return
}
fmt.Printf("created: %s", dbName)
```

After the deferred function is called, the database is dropped.

As an example of how to create many databases:

```go
// set aside database-context connections, say if we want to use them as normal *sql.DB handles
availableDBs := make(map[string]*sql.DB, 0)
for i := 0; i < 10; i++ {
  // important: must be inside loop as dbOptions are updated in place!
  opts := &ivory.DatabaseOptions{
    Host:     "localhost",
    Port:     5555,
    SslMode:  "disable",
    User:     "postgres",
    Password: "rootUserSeriousPassword1",
  }

  // we discard two db handles here, one to the instance, one scoped to the database in the instance teardown() closes
  // database handles and cleans up the created database
  _, dbScopedConn, dbName, teardown, err := ivory.New(ctx, opts, fixtureTable, true, "")
  if err != nil {
    fmt.Println("=> failed to create a database: ", err)
    continue
  }
  defer teardown()
  availableDBs[dbName] = dbScopedConn
}
databasesCreated := ""
for k, _ := range availableDBs {
  databasesCreated += fmt.Sprintf("%s ", k)
}
fmt.Printf("created: %s", databasesCreated)
  // created: _disp_pg_1664125384_q57qcq8qid6a7k7d _disp_pg_1664125384_1yj7i14v7iqauzln _disp_pg_1664125385_galb4ijb...
```

That's it!  No need to act as a database janitor or deal with a slowly growing PostgreSQLinstance.

That said, if your code panics or the context is cancelled before the deferred functions can run,
`FindLikelyAbandonedDBs()` and `DropDB()` are available to you for finding and dropping databases, respectively.

## Development / Contribution

Pull requests or GitHub issues are welcomed.

If you are creating a pull request, please include tests as well as a description of the problem being solved.

If you are opening a GitHub issue, please include the error message and verify that your database is listening to
connections (`connection refused` likely means the wrong database host or port is being specified).
