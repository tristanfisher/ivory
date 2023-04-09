package ivory

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"
)

const createDatabaseInSqlTextTest = "createDatabaseInSqlText"
const sqlTextDbName = "example_created_in_sqlText"

func Test_mightHaveTransaction(t *testing.T) {
	type args struct {
		sqlText string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "obvious",
			args: args{
				sqlText: "begin statement",
			},
			want: true,
		},
		{
			name: "noteSpace",
			args: args{
				sqlText: "begin ",
			},
			want: true,
		},
		{
			name: "dontBeGreedy",
			args: args{
				sqlText: "create table beginnings",
			},
			want: false,
		},
		{
			name: "nothingMatching",
			args: args{
				sqlText: "insert into table pub (table_number, party_size, time) values (3, 7, '2021-10-13 03:38:57.914877')",
			},
			want: false,
		},
		{
			name: "empty",
			args: args{
				sqlText: "",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mightHaveTransaction(tt.args.sqlText); got != tt.want {
				t.Errorf("mightHaveTransaction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDatabaseOptions_DSN(t *testing.T) {
	type fields struct {
		Host                  string
		Port                  int
		Database              string
		Schema                string
		User                  string
		Password              string
		SslMode               string
		ConnectTimeoutSeconds int
		MaxOpenConns          int
		MaxIdleConns          int

		reflectType reflect.Type
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		{
			name: "simpleString",
			fields: fields{
				Host: "localhost",
			},
			want: "host='localhost'",
		},
		{
			name: "digit",
			fields: fields{
				Port: 1024,
			},
			want: "port=1024",
		},
		{
			name: "hostPortIsSpaced",
			fields: fields{
				Host: "localhost",
				Port: 1024,
			},
			want: "host='localhost' port=1024",
		},
		{
			name: "emptyStringSkipped",
			fields: fields{
				Host: "",
			},
			want: "",
		},
		{
			name: "zeroIntSkipped",
			fields: fields{
				Port: 0,
			},
			want: "",
		},
		{
			name: "invalidSslError",
			fields: fields{
				SslMode: "post rock",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "allFields",
			fields: fields{
				Host:                  "localhost",
				Port:                  1024,
				Database:              "exampleDB",
				Schema:                "exampleSchema",
				User:                  "user",
				Password:              "password",
				SslMode:               "disable",
				ConnectTimeoutSeconds: 10,
			},
			want: "host='localhost' port=1024 dbname='exampleDB' search_path='exampleSchema' user='user' password='password' sslmode=disable connect_timeout=10",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			do := &DatabaseOptions{
				Host:                  tt.fields.Host,
				Port:                  tt.fields.Port,
				Database:              tt.fields.Database,
				Schema:                tt.fields.Schema,
				User:                  tt.fields.User,
				Password:              tt.fields.Password,
				SslMode:               tt.fields.SslMode,
				ConnectTimeoutSeconds: tt.fields.ConnectTimeoutSeconds,
				MaxOpenConns:          tt.fields.MaxOpenConns,
				MaxIdleConns:          tt.fields.MaxIdleConns,
				reflectType:           tt.fields.reflectType,
			}

			got, err := do.DSN()
			if (err != nil) != tt.wantErr {
				t.Errorf("DSN() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DSN() = %v, want %v", got, tt.want)
			}
		})
	}
}

// this convenience func creates a string with a temporal and random portion
// as such, we only test len does not exceed our expectation and that repeated runs don't collide on names
func Test_generateDbName(t *testing.T) {

	// e.g. 1634246356
	lenTimeNow := len(strconv.FormatInt(time.Now().Unix(), 10))

	type args struct {
		userProvidedId string
	}
	tests := []struct {
		name    string
		args    args
		wantLen int
	}{
		{
			name: "noIdProvided",
			args: args{
				userProvidedId: "",
			},
			wantLen: PgMaxIdentifierLen - (remainingNameBudget - lenTimeNow),
		},
		{
			name: "IDVeryLong",
			args: args{
				userProvidedId: "aSBhbSByZWFsbHkgc2ljayBvZiBkb2luZyB3ZWIgZGV2ZWxvcG1lbnQhICBpcyBhbnlvbmUgaW50ZXJlc3RpbmcgaGlyaW5nPz8K",
			},
			wantLen: PgMaxIdentifierLen,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateDbName(tt.args.userProvidedId)
			gotLen := len(got)
			if gotLen != tt.wantLen {
				t.Errorf("generateDbName() got len = %v, want len %v, value: %s", gotLen, tt.wantLen, got)
			}
		})
	}
}

// Corresponds to values in docker-compose.yml
func TestNew(t *testing.T) {
	type args struct {
		ctx             context.Context
		opts            *DatabaseOptions
		sqlText         []string
		createDatabase  bool
		customIdPortion string
	}
	tests := []struct {
		name       string
		args       args
		wantDBName string
		wantErr    bool
	}{
		{
			name: "failToConnect",
			args: args{
				ctx: context.TODO(),
				opts: &DatabaseOptions{
					Host:     "test_failToConnect.example.org",
					Port:     99999,
					Database: "abc123",
				},
				sqlText: []string{},
			},
			wantDBName: "", // no db will be created if we can't connect
			wantErr:    true,
		},
		{
			name: "createRandomDB",
			args: args{
				ctx: context.TODO(),
				opts: &DatabaseOptions{
					Host:     "localhost",
					Port:     5555,
					SslMode:  "disable",
					User:     "postgres",
					Password: "rootUserSeriousPassword1",
				},
				createDatabase: true,
				sqlText:        []string{},
			},
			wantDBName: "", // we don't know what it will be
			wantErr:    false,
		},
		{
			name: "honorDBNameCreateDB",
			args: args{
				ctx: context.TODO(),
				opts: &DatabaseOptions{
					Host:     "localhost",
					Port:     5555,
					Database: "flannel",
					SslMode:  "disable",
					User:     "postgres",
					Password: "rootUserSeriousPassword1",
				},
				createDatabase: true,
				sqlText:        []string{"CREATE TABLE foo ( hello CHAR(5));"},
			},
			wantDBName: "flannel",
			wantErr:    false,
		},
		// if the user creates the database in sql text, they must clean up their own database(s)
		{
			name: "createDatabaseInSqlText",
			args: args{
				ctx: context.TODO(),
				opts: &DatabaseOptions{
					Host:     "localhost",
					Port:     5555,
					SslMode:  "disable",
					User:     "postgres",
					Password: "rootUserSeriousPassword1",
				},
				createDatabase: false,
				sqlText:        []string{fmt.Sprintf("CREATE DATABASE %s;", sqlTextDbName)},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbHandleNoBoundDB, dbHandleBound, dbName, tearDown, err := New(tt.args.ctx, tt.args.opts, tt.args.sqlText, tt.args.createDatabase, tt.args.customIdPortion)
			// success of the tear down function should be tested separately
			defer func() {
				// a bit of a hack, but the "create db in the provided text" is a special, advanced case
				// where the caller is responsible.  come up with more elegant behavior if time permits.
				// one naive approach would be allowing the user to include a snippet to run on tearDown
				// that executes before closing connections.
				if tt.name == createDatabaseInSqlTextTest {
					return
				}

				err = tearDown()
				// if we got an err on connection, we could not have created the database
				if (err != nil) != tt.wantErr {
					// stopping before creating more mess
					t.Errorf("Failed to clean up created database: %s. error = %v", dbName, err)
					t.FailNow()
				}
			}()
			// if we received an error, it can be a failure to connect to the db or a failure running sql
			if (err != nil) != tt.wantErr {
				t.Errorf("New() did not expect error. error = %v", err)
				return
			}
			if tt.wantErr {
				return
			}

			if len(tt.wantDBName) > 0 {
				if tt.wantDBName != dbName {
					t.Errorf("New() did not honor the provided DB name. got name = %v, want name = %v", dbName, tt.wantDBName)
					return
				}
			}

			if tt.name == createDatabaseInSqlTextTest {
				// make our own teardown func for this specific test
				err = tearDownFunc(tt.args.ctx, dbHandleNoBoundDB, dbHandleBound, sqlTextDbName)()
				if (err != nil) != tt.wantErr {
					// stopping before creating more mess
					t.Errorf("Failed to clean up created database: %s. error = %v", sqlTextDbName, err)
					t.FailNow()
				}
			}

		})
	}
}

func TestFindLikelyAbandonedDBs(t *testing.T) {

	ctx := context.Background()
	expectedToFind := make([]string, 0)

	testLocalTimeString := strconv.FormatInt(time.Now().Unix(), 10)
	prefix := fmt.Sprintf("TestFLAD_%s", testLocalTimeString)

	// create databases that we want to find later
	// discarding teardown
	dbHandleNoBoundDB1, dbHandleBound1, dbName1, _, err := New(
		ctx,
		&DatabaseOptions{
			Host:     "localhost",
			Port:     5555,
			Database: fmt.Sprintf(prefix + "_1"),
			SslMode:  "disable",
			User:     "postgres",
			Password: "rootUserSeriousPassword1",
		},
		[]string{},
		true,
		"",
	)
	if err != nil {
		t.Errorf("Test setup failed while creating DBs to abandon. error = %v ", err)
		t.FailNow()
	}
	expectedToFind = append(expectedToFind, dbName1)

	dbHandleNoBoundDB2, dbHandleBound2, dbName2, _, err := New(
		ctx,
		&DatabaseOptions{
			Host:     "localhost",
			Port:     5555,
			Database: fmt.Sprintf(prefix + "_2"),
			SslMode:  "disable",
			User:     "postgres",
			Password: "rootUserSeriousPassword1",
		},
		[]string{},
		true,
		"",
	)
	if err != nil {
		t.Errorf("Test setup failed while creating fixture DBs. error = %v ", err)
		t.FailNow()
	}
	expectedToFind = append(expectedToFind, dbName2)

	// close without using our a cleanup function to simulate left work behind
	for _, closeFunc := range []func() error{
		dbHandleNoBoundDB1.Close, dbHandleBound1.Close, dbHandleNoBoundDB2.Close, dbHandleBound2.Close} {
		err = closeFunc()
		if err != nil {
			t.Errorf("Test setup failed closing handle for fixture DB. error = %v ", err)
			t.FailNow()
		}
	}

	// bind a new connection not specific to a database
	noDBOpts := &DatabaseOptions{
		Host:                  "localhost",
		Port:                  5555,
		ConnectTimeoutSeconds: 10,
		SslMode:               "disable",
		User:                  "postgres",
		Password:              "rootUserSeriousPassword1",
	}
	dbHandle, err := Connect(ctx, noDBOpts)
	if err != nil {
		t.Errorf("Test setup failed while creating a new DB handle. error = %v", err)
		t.FailNow()
	}

	defer func(dbHandle *sql.DB) {
		err := dbHandle.Close()
		if err != nil {
			// not really an error in the test, but good to know
			// there is no t.Warn or t.Info
			t.Errorf("Failed to clean up database handle for test.")
		}
	}(dbHandle)

	dbsFound, err := FindLikelyAbandonedDBs(ctx, dbHandle, prefix)
	if err != nil {
		t.Errorf("FindLikelyAbandonedDBs() failed to run. error = %v", err)
		t.FailNow()
		return
	}

	// we could have some databases left over between tests, but it's really unlikely
	if len(dbsFound) != len(expectedToFind) {
		t.Errorf(
			"FindLikelyAbandonedDBs() unexpected number of databases found. got len() = %v, want len = %v ", len(dbsFound), len(expectedToFind))
		if len(dbsFound) == 0 {
			// no point continuing with no work to do
			t.FailNow()
		}
	}

	dbsFoundTable := make(map[string]struct{}, len(dbsFound))
	for _, v := range dbsFound {
		dbsFoundTable[v] = struct{}{}
	}

	for _, expectedDB := range expectedToFind {
		_, ok := dbsFoundTable[expectedDB]
		if !ok {
			t.Errorf("FindLikelyAbandonedDBs() did not find a fixture db. found: %s, missing db: %s", dbsFound, expectedDB)
		}
		// we don't want to use a teardown func in a loop as we'll close the database handle on the first iteration
	}
	_, errSlice := DropDB(ctx, dbHandle, expectedToFind)
	for i, e := range errSlice {
		if e != nil {
			t.Errorf("FindLikelyAbandonedDBs() failed to clean up a test-created database: %s. error = %v", expectedToFind[i], e)
		}
	}
}

// TestNew_UserProvidedSQL specifically tests the functionality of running user-provided SQL
func TestNew_UserProvidedSQL(t *testing.T) {

	ctx := context.Background()
	_, dbHandleBound, dbName, tearDownFunc, err := New(
		ctx,
		&DatabaseOptions{
			Host:     "localhost",
			Port:     5555,
			SslMode:  "disable",
			User:     "postgres",
			Password: "rootUserSeriousPassword1",
		},
		[]string{
			"create schema ivory;",
			"create table ivory.disposable_table ( hello char(5));",
			"insert into ivory.disposable_table (hello) values ('world');",
		},
		true,
		"_test_ups",
	)

	defer func() {
		err := tearDownFunc()
		if err != nil {
			t.Errorf("Failed to clean up created database: %s. error = %v", dbName, err)
		}
	}()

	// test database creation without error
	if err != nil {
		t.Errorf("New() failed to create database for test.  error = %v", err)
		t.FailNow()
	}

	// confirm our table exists
	rows, err := dbHandleBound.QueryContext(ctx, "SELECT hello FROM ivory.disposable_table LIMIT 1;")
	if err != nil {
		t.Errorf("Failed to ")
		t.FailNow()
	}
	var r string

	for rows.Next() {
		err = rows.Scan(&r)
		if err != nil {
			t.Errorf("Failed to scan results.  error = %v", err)
			t.FailNow()
		}
	}

	if r != "world" {
		t.Errorf("New() failed to run sql expressions during setup.  got = %v, want value: %v", r, "world")
	}

}
