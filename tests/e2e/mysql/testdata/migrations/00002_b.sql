-- +goose Up
-- +goose StatementBegin
CREATE TABLE dept_manager (
   emp_no       INT             NOT NULL,
   dept_no      CHAR(4)         NOT NULL,
   from_date    DATE            NOT NULL,
   to_date      DATE            NOT NULL,
   FOREIGN KEY (emp_no)  REFERENCES employee (emp_no)    ON DELETE CASCADE,
   FOREIGN KEY (dept_no) REFERENCES department (dept_no) ON DELETE CASCADE,
   PRIMARY KEY (emp_no,dept_no)
); 
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE dept_manager;
-- +goose StatementEnd
