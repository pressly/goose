package pgdump

import (
	"testing"
)

func TestArgs(t *testing.T) {
	args := Args("mydb", "myuser")
	want := []string{"pg_dump", "--schema-only", "--no-owner", "--no-privileges", "-U", "myuser", "-d", "mydb", "--exclude-table=goose_db_version"}
	if len(args) != len(want) {
		t.Fatalf("got %d args, want %d", len(args), len(want))
	}
	for i := range want {
		if args[i] != want[i] {
			t.Errorf("args[%d] = %q, want %q", i, args[i], want[i])
		}
	}
}

func TestStrip(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "empty input",
			raw:  "",
			want: "",
		},
		{
			name: "only comments",
			raw:  "-- PostgreSQL database dump\n-- Dumped from version 16.1\n",
			want: "",
		},
		{
			name: "SET statements removed",
			raw:  "SET statement_timeout = 0;\nSET lock_timeout = 0;\nSET idle_in_transaction_session_timeout = 0;\n",
			want: "",
		},
		{
			name: "pg_catalog removed",
			raw:  "SELECT pg_catalog.set_config('search_path', '', false);\n",
			want: "",
		},
		{
			name: "DDL preserved",
			raw: `-- PostgreSQL database dump
SET statement_timeout = 0;
SELECT pg_catalog.set_config('search_path', '', false);

CREATE TABLE public.users (
    id integer NOT NULL,
    name text
);

-- PostgreSQL database dump complete
`,
			want: `CREATE TABLE users (
    id integer NOT NULL,
    name text
);
`,
		},
		{
			name: "consecutive blank lines collapsed",
			raw: `CREATE TABLE a (id int);



CREATE TABLE b (id int);
`,
			want: `CREATE TABLE a (id int);

CREATE TABLE b (id int);
`,
		},
		{
			name: "realistic pg_dump output",
			raw: `--
-- PostgreSQL database dump
--

-- Dumped from database version 16.1
-- Dumped by pg_dump version 16.1

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

CREATE TABLE public.users (
    id integer NOT NULL,
    name text NOT NULL,
    email text NOT NULL
);

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);

--
-- PostgreSQL database dump complete
--
`,
			want: `CREATE TABLE users (
    id integer NOT NULL,
    name text NOT NULL,
    email text NOT NULL
);

ALTER TABLE ONLY users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Strip([]byte(tt.raw))
			var gotStr string
			if got != nil {
				gotStr = string(got)
			}
			if gotStr != tt.want {
				t.Errorf("Strip() mismatch:\n--- got ---\n%s\n--- want ---\n%s", gotStr, tt.want)
			}
		})
	}
}
