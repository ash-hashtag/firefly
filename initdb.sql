
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE OR REPLACE FUNCTION ulid_to_epoch_ms(input uuid) RETURNS bigint
    LANGUAGE plpgsql
    AS $$
BEGIN
    return
        (  'x'
        || '0000'
        || SUBSTRING(input::text,  7, 2)
        || SUBSTRING(input::text,  5, 2)
        || SUBSTRING(input::text,  3, 2)
        || SUBSTRING(input::text,  1, 2)
        || SUBSTRING(input::text, 12, 2)
        || SUBSTRING(input::text, 10, 2)
        )::bit(64)::bigint AS int8_val;
END
$$;

CREATE OR REPLACE FUNCTION extract_ulid_timestamp(uuid_input UUID) RETURNS BIGINT AS $$
DECLARE
    uuid_hex TEXT;
    ulid_timestamp BIGINT;
BEGIN
    -- Convert UUID to hex text without dashes
    uuid_hex := replace(uuid_input::TEXT, '-', '');
    
    -- Extract the first 12 characters (6 bytes in hex) and convert to bigint
    ulid_timestamp := ('x' || substring(uuid_hex from 1 for 12))::BIT(48)::BIGINT;

    RETURN ulid_timestamp;
END;
$$ LANGUAGE plpgsql;


CREATE OR REPLACE FUNCTION generate_ulid() RETURNS uuid
    LANGUAGE plpgsql
    AS $$
DECLARE
  timestamp  BYTEA = E'\\000\\000\\000\\000\\000\\000';
  unix_time  BIGINT;
BEGIN
    unix_time = (EXTRACT(EPOCH FROM NOW()) * 1000)::BIGINT;

    timestamp = SET_BYTE(timestamp, 0, (unix_time >> 40)::BIT(8)::INTEGER);
    timestamp = SET_BYTE(timestamp, 1, (unix_time >> 32)::BIT(8)::INTEGER);
    timestamp = SET_BYTE(timestamp, 2, (unix_time >> 24)::BIT(8)::INTEGER);
    timestamp = SET_BYTE(timestamp, 3, (unix_time >> 16)::BIT(8)::INTEGER);
    timestamp = SET_BYTE(timestamp, 4, (unix_time >> 8)::BIT(8)::INTEGER);
    timestamp = SET_BYTE(timestamp, 5, unix_time::BIT(8)::INTEGER);

    RETURN encode( timestamp || gen_random_bytes(10) ,'hex')::uuid;
END
$$;


CREATE TABLE IF NOT EXISTS groupchats (
    id SERIAL NOT NULL PRIMARY KEY,
    created_by VARCHAR NOT NULL,
    name VARCHAR NOT NULL,
);



CREATE TABLE IF NOT EXISTS groupchannels (
    chanId INTEGER NOT NULL,
    grpId INTEGER REFERENCES groupchats (id) NOT NULL,
    name VARCHAR NOT NULL,
    chanType SMALLINT NOT NULL,

    PRIMARY KEY (grpId, chanId)
);

-- perms
-- 1 create channel
-- 2 create message
-- 4 invite members
-- 8 kick members




CREATE TABLE IF NOT EXISTS groupmembers (
    grpId INTEGER REFERENCES groupchats (id) NOT NULL,
    chanId INTEGER NOT NULL,
    uname VARCHAR NOT NULL,
    role INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (grpId, chanId, uname)
);

CREATE TABLE IF NOT EXISTS grouproles (
    grpId INTEGER REFERENCES groupchats (id) NOT NULL,
    role INTEGER NOT NULL,
    name VARCHAR NOT NULL,
    perms INTEGER NOT NULL
    -- bitset permissions

    PRIMARY KEY (grpId, role)
);


CREATE INDEX IF NOT EXISTS uname_idx_on_groupmembers ON groupmembers (uname);


CREATE TABLE IF NOT EXISTS groupmessages (
    id UUID NOT NULL PRIMARY KEY DEFAULT generate_ulid();
    grpId INTEGER NOT NULL,
    chanId INTEGER NOT NULL,
    msg TEXT NOT NULL,
    by VARCHAR NOT NULL REFERENCES groupmembers ( uname )
);

CREATE INDEX IF NOT EXISTS grp_chan_index ON groupmessages (grpId, chanId);



CREATE OR REPLACE FUNCTION create_group_channel (groupId VARCHAR, chanType SMALLINT, chanName VARCHAR, createdBy VARCHAR) RETURNS INTEGER AS
$$
BEGIN
 IF NOT EXISTS (
        SELECT 1
        FROM groupmembers gm
        JOIN grouproles gr ON gr.grpId = gm.grpId AND gr.role = gm.role
        WHERE gm.grpId = groupId
          AND gm.chanId = channelId
          AND gm.uname = createdBy
          AND (gr.perms & 1) = 1
    ) THEN
        RAISE NOTICE 'adder either not in channel/group or lacks permission';
    END IF;



    INSERT INTO groupchannels (grpId, chanId, name, chanType)
    SELECT groupId, COALESCE((
        SELECT MAX(chanId) FROM groupchannels WHERE grpId = groupId
    ), 0) + 1, chanName, chanType
    RETURNING chanId;

END
$$ LANGUAGE plpgsql;


CREATE OR REPLACE FUNCTION add_group_member (username VARCHAR, groupId INTEGER, channelId INTEGER, urole INTEGER, addedBy VARCHAR) RETURNS VOID AS $$

DECLARE
adderPerms INTEGER;
addeePerms INTEGER;
BEGIN

    SELECT COALESCE(perms, -1) INTO adderPerms FROM groupmembers gm
    JOIN grouproles gr ON gr.grpId = gm.grpId AND gr.role = gm.role
    WHERE gm.grpId = groupId AND gm.chanId = channelId AND gm.uname = addedBy; 

    IF adderPerms = -1 OR adderPerms & 4 <> 4 THEN
        RAISE NOTICE 'adder either not in channel/group or lacks permissions'
    END IF;


    SELECT COALESCE(perms, -1) INTO addeePerms FROM grouproles WHERE role = urole;

    IF addeePerms = -1 OR addeePerms & adderPerms <> addeePerms THEN
        RAISE NOTICE 'addee role does not exist, or adder doesnot have enough authority to assign the role'
    END IF;

    INSERT INTO groupmembers (grpId, chanId, uname, role) VALUES (groupId, channelId, username, urole);

END
$$ LANGUAGE plpgsql;


CREATE OR REPLACE FUNCTION kick_group_member (username VARCHAR, groupId INTEGER, channelId INTEGER, kickedBy VARCHAR) RETURNS VOID AS $$
BEGIN

 IF NOT EXISTS (
        SELECT 1
        FROM groupmembers gm
        JOIN grouproles gr ON gr.grpId = gm.grpId AND gr.role = gm.role
        WHERE gm.grpId = groupId
          AND gm.chanId = channelId
          AND gm.uname = kickedBy
          AND (gr.perms & 8) = 8
    ) THEN
        RAISE NOTICE 'kicker either not in channel/group or lacks permission';
    END IF;


    DELETE FROM groupmembers WHERE grpId = groupId AND chanId = channelId AND uname = username;

END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION check_user_can_message() RETURNS TRIGGER AS $$
BEGIN

 IF NOT EXISTS (
        SELECT 1
        FROM groupmembers gm
        JOIN grouproles gr ON gr.grpId = gm.grpId AND gr.role = gm.role
        WHERE gm.grpId = NEW.grpId
          AND gm.chanId = NEW.chanId
          AND gm.uname = NEW.by
          AND (gr.perms & 2) = 2
    ) THEN
        RAISE NOTICE 'messeger either not in channel/group or lacks permission';
    END IF;

    RETURN NEW;

END;
$$ LANGUAGE plpgsql;


CREATE TRIGGER enforce_user_message_permissions
BEFORE INSERT ON posts
FOR EACH ROW EXECUTE FUNCTION check_user_can_message();
