package main

const CreateTablesIfNotExists = `
	CREATE TABLE IF NOT EXISTS cams
	(
		id BLOB PRIMARY KEY,
		addr STRING,
		adminka STRING,
		stream STRING,
		manufacturer STRING,
		country STRING
	);

	CREATE TABLE IF NOT EXISTS visitors (
		id STRING PRIMARY KEY,
		last_seen INTEGER
	);

	CREATE TABLE IF NOT EXISTS cams_visitors
	(	
		visitor_id STRING,
		cam_id INTEGER,
		PRIMARY KEY (visitor_id, cam_id),
		FOREIGN KEY (cam_id) REFERENCES cams (id) ON DELETE CASCADE,
		FOREIGN KEY (visitor_id) REFERENCES visitors (id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS reactions
	(
		like INTEGER,
		visitor_id STRING,
		cam_id BLOB,
		PRIMARY KEY (visitor_id, cam_id),
		FOREIGN KEY (cam_id) REFERENCES videos (id) ON DELETE CASCADE,
		FOREIGN KEY (visitor_id) REFERENCES visitors (id) ON DELETE CASCADE
	);
`
