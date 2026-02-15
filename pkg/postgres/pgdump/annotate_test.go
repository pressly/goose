package pgdump

import (
	"testing"
)

func TestAnnotate(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{
			name: "empty input",
			data: "",
			want: "",
		},
		{
			name: "simple statements not wrapped",
			data: `CREATE TABLE users (
    id integer NOT NULL,
    name text
);

ALTER TABLE ONLY users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);
`,
			want: `-- +goose Up
CREATE TABLE users (
    id integer NOT NULL,
    name text
);

ALTER TABLE ONLY users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);
`,
		},
		{
			name: "function with $$ gets wrapped",
			data: `CREATE TABLE users (
    id integer NOT NULL
);

CREATE FUNCTION parse_timestamp(ts text)
    RETURNS timestamp with time zone
    LANGUAGE plpgsql
    IMMUTABLE
    AS $$
BEGIN
    RETURN ts::timestamptz;
END
$$;

ALTER TABLE ONLY users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);
`,
			want: `-- +goose Up
CREATE TABLE users (
    id integer NOT NULL
);

-- +goose StatementBegin
CREATE FUNCTION parse_timestamp(ts text)
    RETURNS timestamp with time zone
    LANGUAGE plpgsql
    IMMUTABLE
    AS $$
BEGIN
    RETURN ts::timestamptz;
END
$$;
-- +goose StatementEnd

ALTER TABLE ONLY users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);
`,
		},
		{
			name: "multiple functions wrapped independently",
			data: `CREATE FUNCTION fn_a()
    RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    NULL;
END
$$;

CREATE FUNCTION fn_b()
    RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    NULL;
END
$$;
`,
			want: `-- +goose Up
-- +goose StatementBegin
CREATE FUNCTION fn_a()
    RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    NULL;
END
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION fn_b()
    RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    NULL;
END
$$;
-- +goose StatementEnd
`,
		},
		{
			name: "function with blank lines inside body",
			data: `CREATE FUNCTION tg_change()
    RETURNS TRIGGER
    LANGUAGE plpgsql
    AS $$
BEGIN
    IF (TG_OP = 'INSERT') THEN
        INSERT INTO logs (event) VALUES ('CREATE');

    ELSIF (TG_OP = 'DELETE') THEN
        INSERT INTO logs (event) VALUES ('DELETE');

    END IF;
    RETURN NULL;
END;
$$;

CREATE TABLE logs (
    event text NOT NULL
);
`,
			want: `-- +goose Up
-- +goose StatementBegin
CREATE FUNCTION tg_change()
    RETURNS TRIGGER
    LANGUAGE plpgsql
    AS $$
BEGIN
    IF (TG_OP = 'INSERT') THEN
        INSERT INTO logs (event) VALUES ('CREATE');

    ELSIF (TG_OP = 'DELETE') THEN
        INSERT INTO logs (event) VALUES ('DELETE');

    END IF;
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

CREATE TABLE logs (
    event text NOT NULL
);
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Annotate([]byte(tt.data))
			var gotStr string
			if got != nil {
				gotStr = string(got)
			}
			if gotStr != tt.want {
				t.Errorf("Annotate() mismatch:\n--- got ---\n%s\n--- want ---\n%s", gotStr, tt.want)
			}
		})
	}
}
