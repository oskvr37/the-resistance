#!lua name=room

redis.register_function("room_refresh", function(keys, args)
    -- 1. Check if the room actually exists
    if redis.call("EXISTS", keys[2]) == 0 then
        return {
            err = "ROOM_NOT_FOUND"
        }
    end

    -- 2. Define the TTL (10 minutes as per your 600s)
    -- Tip: You could also pass this via args[1] if you want it dynamic
    local ttl = 600

    -- 3. Iterate through all provided keys and refresh them
    for i, key in ipairs(keys) do
        -- Only expire if the key actually exists to avoid cluttering 
        -- keyspaces with empty structures
        if redis.call("EXISTS", key) == 1 then
            redis.call("EXPIRE", key, ttl)
        end
    end

    return "OK"
end)

redis.register_function("room_create", function(keys, args)
    local KEY_USER_ROOM = keys[1]
    local KEY_ROOM_USERS = keys[2]
    local KEY_ROOM_EVENTS = keys[3]
    local KEY_ROOM = keys[4]
    local KEY_JOIN_CODE = keys[5]
    local KEY_USER = keys[6]
    local KEY_ROOM_PRESENCE = keys[7]

    local ID_USER = args[1]
    local ID_ROOM = args[2]
    local JOIN_CODE = args[3]
    local USERNAME = args[4]
    local AVATAR = args[5]

    if redis.call("EXISTS", KEY_USER_ROOM) == 1 then
        return {"ERR", "ALREADY_IN_ROOM"}
    end

    if redis.call("EXISTS", KEY_ROOM) == 1 then
        return {"ERR", "ROOM_ID_COLLISION"}
    end

    if redis.call("EXISTS", KEY_JOIN_CODE) == 1 then
        return {"ERR", "JOIN_CODE_COLLISION"}
    end

    -- create room
    redis.call("HSET", KEY_ROOM, "owner", ID_USER, "join_code", JOIN_CODE, "status", "LOBBY", "max_players", 10,
        "game_pace", "STANDARD")
    redis.call("EXPIRE", KEY_ROOM, 600)

    -- map room join code
    redis.call("SET", KEY_JOIN_CODE, ID_ROOM)
    redis.call("EXPIRE", KEY_JOIN_CODE, 600)

    -- add user to room
    redis.call("SET", KEY_USER_ROOM, ID_ROOM)
    redis.call("EXPIRE", KEY_USER_ROOM, 600)

    local user_data = cjson.encode({
        username = USERNAME,
        avatar = AVATAR
    })
    redis.call("HSET", KEY_ROOM_USERS, ID_USER, user_data)
    redis.call("EXPIRE", KEY_ROOM_USERS, 600)

    -- set user presence
    local TIME = redis.call("TIME")
    redis.call("ZADD", KEY_ROOM_PRESENCE, TIME[1], ID_USER)
    redis.call("EXPIRE", KEY_ROOM_PRESENCE, 600)

    redis.call("PUBLISH", KEY_ROOM_EVENTS, "ROOM_CREATE")

    return {"OK", "OK"}
end)

redis.register_function("room_join", function(keys, args)
    local KEY_USER_ROOM = keys[1]
    local KEY_ROOM_USERS = keys[2]
    local KEY_ROOM_EVENTS = keys[3]
    local KEY_ROOM = keys[4]
    local KEY_USER = keys[5]
    local KEY_ROOM_PRESENCE = keys[6]

    local ID_USER = args[1]
    local ID_ROOM = args[2]
    local USERNAME = args[3]
    local AVATAR = args[4]
    local NOW = args[5]

    if redis.call("EXISTS", KEY_USER_ROOM) == 1 then
        return {"ERR", "ALREADY_IN_ROOM"}
    end

    if redis.call("EXISTS", KEY_ROOM) == 0 then
        return {"ERR", "ROOM_NOT_FOUND"}
    end

    local room_status = redis.call("HGET", KEY_ROOM, "status")
    if room_status ~= "LOBBY" then
        return {"ERR", "ROOM_ALREADY_STARTED"}
    end

    local current_count = redis.call("HLEN", KEY_ROOM_USERS)
    local max_players_raw = redis.call("HGET", KEY_ROOM, "max_players")

    if not max_players_raw then
        return {"ERR", "INTERNAL"}
    end

    local max_players = tonumber(max_players_raw)

    if current_count >= max_players then
        return {"ERR", "ROOM_FULL"}
    end

    -- add user to room
    redis.call("SET", KEY_USER_ROOM, ID_ROOM)
    redis.call("EXPIRE", KEY_USER_ROOM, 600)

    local user_data = cjson.encode({
        username = USERNAME,
        avatar = AVATAR
    })
    redis.call("HSET", KEY_ROOM_USERS, ID_USER, user_data)
    redis.call("EXPIRE", KEY_ROOM_USERS, 600)

    -- set user presence
    local TIME = redis.call("TIME")
    redis.call("ZADD", KEY_ROOM_PRESENCE, TIME[1], ID_USER)
    redis.call("EXPIRE", KEY_ROOM_PRESENCE, 600)

    -- check if room has an owner
    if not redis.call("HGET", KEY_ROOM, "owner") then
        local new_owner = redis.call("HRANDFIELD", KEY_ROOM_USERS)
        redis.call("HSET", KEY_ROOM, "owner", new_owner)
    end

    redis.call("PUBLISH", KEY_ROOM_EVENTS, "ROOM_JOIN")

    return {"OK", "OK"}
end)

redis.register_function("room_leave", function(keys, args)
    local KEY_USER_ROOM = keys[1]
    local KEY_ROOM_USERS = keys[2]
    local KEY_ROOM_EVENTS = keys[3]
    local KEY_ROOM = keys[4]
    local KEY_ROOM_PRESENCE = keys[5]

    local ID_USER = args[1]
    local ID_ROOM = args[2]

    local user_room_id = redis.call("GET", KEY_USER_ROOM)
    if not user_room_id or user_room_id ~= ID_ROOM then
        return {"ERR", "NOT_IN_THIS_ROOM"}
    end

    if redis.call("EXISTS", KEY_ROOM) == 0 then
        -- room already gone
        redis.call("DEL", KEY_USER_ROOM)
        return {"OK", "NO_CONTENT"}
    end

    local room_status = redis.call("HGET", KEY_ROOM, "status")
    if room_status ~= "LOBBY" then
        redis.call("EXPIRE", KEY_USER_ROOM, 3600)
        return {"ERR", "CANT_LEAVE_STARTED_ROOM"}
    end

    -- remove user from room
    redis.call("DEL", KEY_USER_ROOM)
    redis.call("HDEL", KEY_ROOM_USERS, ID_USER)
    redis.call("ZREM", KEY_ROOM_PRESENCE, ID_USER)

    -- select new owner if leaving user was the owner
    if redis.call("HGET", KEY_ROOM, "owner") == ID_USER then
        if redis.call("HLEN", KEY_ROOM_USERS) >= 1 then
            local new_owner = redis.call("HRANDFIELD", KEY_ROOM_USERS)
            redis.call("HSET", KEY_ROOM, "owner", new_owner)
        else
            redis.call("HDEL", KEY_ROOM, "owner")
        end
    end

    redis.call("PUBLISH", KEY_ROOM_EVENTS, "ROOM_LEAVE")

    return {"OK", "NO_CONTENT"}
end)

redis.register_function("room_heartbeat", function(keys, args)
    local KEY_ROOM_PRESENCE = keys[1]
    local KEY_ROOM_EVENTS = keys[2]
    local KEY_USER_ROOM = keys[3]
    local KEY_ROOM = keys[4]
    local KEY_ROOM_USERS = keys[5]
    local KEY_USER = keys[6]

    local USER_ID = args[1]
    local ROOM_ID = args[2]

    if redis.call("GET", KEY_USER_ROOM) ~= ROOM_ID then
        return {"ERR", "NOT_IN_THIS_ROOM"}
    end

    redis.call("EXPIRE", KEY_USER_ROOM, 600)

    local TIME_NOW = redis.call("TIME")
    local added = redis.call("ZADD", KEY_ROOM_PRESENCE, TIME_NOW[1], USER_ID)

    local removed = redis.call("ZREMRANGEBYSCORE", KEY_ROOM_PRESENCE, "-inf", TIME_NOW[1] - 15)

    if added > 0 or removed > 0 then
        redis.call("PUBLISH", KEY_ROOM_EVENTS, "ROOM_HEARTBEAT")
    end

    return {"OK", "NO_CONTENT"}
end)

redis.register_function("room_user_kick", function(keys, args)
    local KEY_ROOM = keys[1]
    local KEY_ROOM_USERS = keys[2]
    local KEY_ROOM_PRESENCE = keys[3]
    local KEY_ROOM_EVENTS = keys[4]
    local KEY_KICKED_USER_ROOM = keys[5]

    local OWNER_ID = args[1]
    local KICKED_ID = args[2]

    if redis.call("EXISTS", KEY_ROOM) == 0 then
        return {"ERR", "ROOM_NOT_FOUND"}
    end

    local room_owner = redis.call("HGET", KEY_ROOM, "owner")
    if room_owner ~= OWNER_ID then
        return {"ERR", "NOT_ROOM_OWNER"}
    end

    if OWNER_ID == KICKED_ID then
        return {"ERR", "CANT_KICK_YOURSELF"}
    end

    local room_status = redis.call("HGET", KEY_ROOM, "status")
    if room_status == "PLAYING" then
        return {"ERR", "CANT_KICK_DURING_GAME"}
    end

    local is_user_removed = redis.call("HDEL", KEY_ROOM_USERS, KICKED_ID)
    if is_user_removed == 0 then
        return {"ERR", "KICKED_USER_NOT_IN_ROOM"}
    end

    -- remove user from room
    redis.call("DEL", KEY_KICKED_USER_ROOM)
    redis.call("ZREM", KEY_ROOM_PRESENCE, KICKED_ID)

    redis.call("PUBLISH", KEY_ROOM_EVENTS, "ROOM_USER_KICK")
    return {"OK", "NO_CONTENT"}
end)

redis.register_function("room_settings", function(keys, args)
    local KEY_ROOM = keys[1]
    local KEY_ROOM_EVENTS = keys[2]
    local KEY_ROOM_USERS = keys[3]

    local OWNER_ID = args[1]
    local SETTING_MAX_PLAYERS = args[2]
    local SETTING_GAME_PACE = args[3]

    local room_data = redis.call("HMGET", KEY_ROOM, "owner", "status")
    local current_owner = room_data[1]
    local room_status = room_data[2]

    if not current_owner then
        return {"ERR", "ROOM_NOT_FOUND"}
    end
    if current_owner ~= OWNER_ID then
        return {"ERR", "NOT_ROOM_OWNER"}
    end
    if room_status == "PLAYING" then
        return {"ERR", "CANT_CHANGE_SETTINGS_DURING_GAME"}
    end

    -- 1. Only validate Max Players if the user is actually trying to change it
    if SETTING_MAX_PLAYERS ~= "" then
        local max_players_num = tonumber(SETTING_MAX_PLAYERS)

        -- Guard against non-numeric strings
        if not max_players_num then
            return {"ERR", "MAX_PLAYERS_MUST_BE_A_NUMBER"}
        end

        local current_member_count = redis.call("HLEN", KEY_ROOM_USERS)
        if max_players_num < current_member_count then
            return {"ERR", "CANT_DECREASE_PLAYER_COUNT"}
        end

        redis.call("HSET", KEY_ROOM, "max_players", SETTING_MAX_PLAYERS)
    end

    -- 2. Update Pace if provided
    if SETTING_GAME_PACE ~= "" then
        redis.call("HSET", KEY_ROOM, "game_pace", SETTING_GAME_PACE)
    end

    redis.call("PUBLISH", KEY_ROOM_EVENTS, "ROOM_SETTINGS")

    return {"OK", "NO_CONTENT"}
end)
