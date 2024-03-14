-- +goose up

-- +goose statementbegin
CREATE OR REPLACE PROCEDURE insert_repository(
    IN p_repo_full_name VARCHAR(255),
    IN p_owner_name VARCHAR(255),
    IN p_owner_type VARCHAR(20)
)
BEGIN
    DECLARE v_owner_id BIGINT;
    DECLARE v_repo_id BIGINT;

    -- Check if the owner already exists
    SELECT owner_id INTO v_owner_id
    FROM owners
    WHERE owner_name = p_owner_name AND owner_type = p_owner_type;

    -- If the owner does not exist, insert a new owner
    IF v_owner_id IS NULL THEN
        INSERT INTO owners (owner_name, owner_type)
        VALUES (p_owner_name, p_owner_type);
        
        SET v_owner_id = LAST_INSERT_ID();
    END IF;

    -- Insert the repository using the obtained owner_id
    INSERT INTO repos (repo_full_name, repo_owner_id)
    VALUES (p_repo_full_name, v_owner_id);

    -- No explicit return needed in procedures

END;
-- +goose statementend

-- +goose down
DROP PROCEDURE IF EXISTS insert_repository;
