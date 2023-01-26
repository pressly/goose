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
