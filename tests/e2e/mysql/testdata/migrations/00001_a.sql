-- See https://github.com/bytebase/employee-sample-database
-- for more details about the schema.

-- +goose Up
-- +goose StatementBegin
CREATE TABLE employee (
    emp_no      INT             NOT NULL,
    birth_date  DATE            NOT NULL,
    first_name  VARCHAR(14)     NOT NULL,
    last_name   VARCHAR(16)     NOT NULL,
    gender      ENUM ('M','F')  NOT NULL,    
    hire_date   DATE            NOT NULL,
    PRIMARY KEY (emp_no)
);

CREATE TABLE department (
    dept_no     CHAR(4)         NOT NULL,
    dept_name   VARCHAR(40)     NOT NULL,
    PRIMARY KEY (dept_no),
    UNIQUE  KEY (dept_name)
);

-- +goose StatementEnd


-- +goose Down
DROP TABLE employee;
DROP TABLE department;
