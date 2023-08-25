-- +goose Up
-- +goose StatementBegin
create table t ( id int );
update rows set value = now() -- missing semicolon. valid statement because wrapped in goose annotation, but will fail when executed.
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS t;
