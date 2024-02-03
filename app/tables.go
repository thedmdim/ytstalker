package main

const CreateTablesIfNotExists = `
	CREATE TABLE IF NOT EXISTS videos
	(
		id STRING PRIMARY KEY,
		ip STRING,
		adminka STRING,
		stream STRING,
		country string
	);

	CREATE TABLE IF NOT EXISTS visitors (
		id STRING PRIMARY KEY,
		last_seen INTEGER
	);

	CREATE TABLE IF NOT EXISTS videos_visitors
	(	
		visitor_id STRING,
		video_id STRING,
		PRIMARY KEY (visitor_id, video_id),
		FOREIGN KEY (video_id) REFERENCES videos (id) ON DELETE CASCADE,
		FOREIGN KEY (visitor_id) REFERENCES visitors (id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS reactions
	(
		cool INTEGER,
		visitor_id STRING,
		video_id STRING,
		PRIMARY KEY (visitor_id, video_id),
		FOREIGN KEY (video_id) REFERENCES videos (id) ON DELETE CASCADE,
		FOREIGN KEY (visitor_id) REFERENCES visitors (id) ON DELETE CASCADE
	);
`
