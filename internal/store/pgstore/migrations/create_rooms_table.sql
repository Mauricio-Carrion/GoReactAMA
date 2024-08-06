CREATE TABLE IF NOT EXISTS rooms (
  "id" uuid PRIMARY KEY NOT NULL default gen_random_uuid(), 
  "theme" VARCHAR(255) NOT NULL
);