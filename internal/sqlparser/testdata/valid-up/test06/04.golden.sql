-- 1 this comment will be preserved
  -- 2 this comment will be preserved


CREATE FUNCTION do_something(sql TEXT) RETURNS INTEGER AS $$
BEGIN
  -- initiate technology
  PERFORM something_or_other(sql);

  -- increase technology
  PERFORM some_other_thing(sql);

  -- technology was successful
  RETURN 1;
END;
$$ LANGUAGE plpgsql;

-- 3 this comment will be preserved
  -- 4 this comment will be preserved
