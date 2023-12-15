    
-- +goose Up
-- +goose StatementBegin
CREATE INDEX ON public.users (user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS users_user_id_idx;
-- +goose StatementEnd
