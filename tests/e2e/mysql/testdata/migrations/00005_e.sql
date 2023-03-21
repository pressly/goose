-- +goose Up
CREATE TABLE salary (
    emp_no      INT             NOT NULL,
    amount      INT             NOT NULL,
    from_date   DATE            NOT NULL,
    to_date     DATE            NOT NULL,
    FOREIGN KEY (emp_no) REFERENCES employee (emp_no) ON DELETE CASCADE,
    PRIMARY KEY (emp_no, from_date)
) ;

-- +goose Down
DROP TABLE salary;
