CREATE TABLE servers (
    id INTEGER PRIMARY KEY,
    host TEXT NOT NULL
);

CREATE TABLE files (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    uploaded_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE file_parts (
    file_id UUID REFERENCES files (id),
    server_id INTEGER REFERENCES servers (id),
    part_num INTEGER,
    part_size BIGINT
);

CREATE INDEX file_parts_file_id_idx ON file_parts (file_id);
CREATE INDEX file_parts_server_id_idx ON file_parts (server_id);