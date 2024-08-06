CREATE TABLE IF NOT EXISTS messages (
  "id" uuid PRIMARY KEY NOT NULL default gen_random_uuid(), 
  "room_id" uuid NOT NULL,
  "message" VARCHAR(255) NOT NULL,
  "reaction_count" BIGINT NOT NULL default 0,
  "answered" BOOLEAN NOT NULL default false,

  FOREIGN KEY (room_id) REFERENCES rooms(id)  
);