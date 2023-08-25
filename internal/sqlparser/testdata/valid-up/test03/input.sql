-- +goose Up
-- +goose StatementBegin
CREATE FUNCTION cs_update_referrer_type_proc() RETURNS INTEGER AS '
DECLARE
    referrer_keys RECORD;  -- Declare a generic record to be used in a FOR
    a_output varchar(4000);
BEGIN 
    a_output := ''CREATE FUNCTION cs_find_referrer_type(varchar,varchar,varchar) 
                  RETURNS VARCHAR AS '''' 
                     DECLARE 
                         v_host ALIAS FOR $1; 
                         v_domain ALIAS FOR $2; 
                         v_url ALIAS FOR $3;
                     BEGIN ''; 

-- 
-- Notice how we scan through the results of a query in a FOR loop
-- using the FOR <record> construct.
--

    FOR referrer_keys IN SELECT * FROM cs_referrer_keys ORDER BY try_order LOOP
        a_output := a_output || '' IF v_'' || referrer_keys.kind || '' LIKE '''''''''' 
                 || referrer_keys.key_string || '''''''''' THEN RETURN '''''' 
                 || referrer_keys.referrer_type || ''''''; END IF;''; 
    END LOOP; 
  
    a_output := a_output || '' RETURN NULL; END; '''' LANGUAGE ''''plpgsql'''';''; 
 
    -- This works because we are not substituting any variables
    -- Otherwise it would fail. Look at PERFORM for another way to run functions
    
    EXECUTE a_output; 
END; 
' LANGUAGE 'plpgsql'; -- This comment WILL BE preserved.
-- And so will this one.
-- +goose StatementEnd

-- +goose Down
