-- +goose Up
INSERT INTO clickstream VALUES ('customer1', '2021-10-02', 'add_to_cart', 'US', 568239 ); 

INSERT INTO clickstream (customer_id, time_stamp, click_event_type) VALUES ('customer2', '2021-10-30', 'remove_from_cart' );

INSERT INTO clickstream (* EXCEPT(country_code)) VALUES ('customer3', '2021-11-07', 'checkout', 307493 );

-- +goose Down
