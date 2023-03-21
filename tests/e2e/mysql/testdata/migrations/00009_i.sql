-- +goose Up
-- +goose StatementBegin
-- 
create procedure show_department()
modifies sql data
begin
    DROP TABLE IF EXISTS department_max_date;
    DROP TABLE IF EXISTS department_people;
    CREATE TEMPORARY TABLE department_max_date
    (
        emp_no int not null primary key,
        dept_from_date date not null,
        dept_to_date  date not null, # bug#320513
        KEY (dept_from_date, dept_to_date)
    );
    INSERT INTO department_max_date
    SELECT
        emp_no, max(from_date), max(to_date)
    FROM
        dept_emp
    GROUP BY
        emp_no;

    CREATE TEMPORARY TABLE department_people
    (
        emp_no int not null,
        dept_no char(4) not null,
        primary key (emp_no, dept_no)
    );

    insert into department_people
    select dmd.emp_no, dept_no
    from
        department_max_date dmd
        inner join dept_emp de
            on dmd.dept_from_date=de.from_date
            and dmd.dept_to_date=de.to_date
            and dmd.emp_no=de.emp_no;
    SELECT
        dept_no,dept_name,manager, count(*)
        from v_full_department
            inner join department_people using (dept_no)
        group by dept_no;
        # with rollup;
    DROP TABLE department_max_date;
    DROP TABLE department_people;
end;
-- +goose StatementEnd

-- +goose Down
DROP PROCEDURE show_department;
