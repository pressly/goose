-- +goose Up
CREATE MATERIALIZED VIEW IF NOT EXISTS matview_stargazers_day AS
	SELECT
		t.*,
		repo_full_name,
        repo_owner_id
	FROM (
	SELECT
		date_trunc('day', stargazer_starred_at)::date AS stars_day,
		count(*) AS total,
		stargazer_repo_id
	FROM
		stargazers
	GROUP BY
		stars_day,
		stargazer_repo_id) AS t
	JOIN repos ON stargazer_repo_id = repo_id
ORDER BY
	stars_day;

CREATE UNIQUE INDEX ON matview_stargazers_day (stargazer_repo_id, stars_day, repo_owner_id, repo_full_name);

REFRESH MATERIALIZED VIEW CONCURRENTLY matview_stargazers_day WITH DATA;

-- +goose Down
DROP MATERIALIZED VIEW IF EXISTS matview_stargazers_day;