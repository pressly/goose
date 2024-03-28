-- +goose up
-- +goose statementbegin
CREATE OR REPLACE FUNCTION insert_repository(
    p_repo_full_name TEXT,
    p_owner_name TEXT,
    p_owner_type OWNER_TYPE
) RETURNS VOID AS $$
DECLARE
    v_owner_id BIGINT;
    v_repo_id BIGINT;
BEGIN
    -- Check if the owner already exists
    SELECT owner_id INTO v_owner_id
    FROM owners
    WHERE owner_name = p_owner_name AND owner_type = p_owner_type;

    -- If the owner does not exist, insert a new owner
    IF v_owner_id IS NULL THEN
        INSERT INTO owners (owner_name, owner_type)
        VALUES (p_owner_name, p_owner_type)
        RETURNING owner_id INTO v_owner_id;
    END IF;

    -- Insert the repository using the obtained owner_id
    INSERT INTO repos (repo_full_name, repo_owner_id)
    VALUES (p_repo_full_name, v_owner_id)
    RETURNING repo_id INTO v_repo_id;

    -- Commit the transaction
    COMMIT;
END;
$$ LANGUAGE plpgsql;
-- +goose statementend

-- +goose down
DROP FUNCTION IF EXISTS insert_repository(TEXT, TEXT, OWNER_TYPE);
