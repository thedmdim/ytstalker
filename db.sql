CREATE TABLE IF NOT EXISTS videos
	(
		id STRING UNIQUE,
		uploaded INTEGER,
		title STRING,
		views INTEGER,
		vertical INTEGER,
		category INTEGER
	);

CREATE TABLE IF NOT EXISTS videos_visitors
(
    visitor_id STRING,
    video_id STRING,

    FOREIGN KEY (video_id) REFERENCES videos (id) ON DELETE CASCADE,
    FOREIGN KEY (visitor_id) REFERENCES visitors (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS visitors (
    id STRING UNIQUE,
    last_seen INTEGER
);