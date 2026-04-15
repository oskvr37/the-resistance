#!lua name=game

-- Helper: Fisher-Yates shuffle
local function shuffle(t)
    local n = #t
    for i = n, 2, -1 do
        local j = math.random(i)
        t[i], t[j] = t[j], t[i]
    end
    return t
end

-- Helper: Create a copy of a table
local function copy_table(t)
    local u = {}
    for k, v in ipairs(t) do
        u[k] = v
    end
    return u
end

-- Helper: Calculate expiration time based on pace and phase
local function get_phase_expiration(pace, phase)
    local pace_durations = {
        ["RELAXED"] = {
            NOMINATION = 300,
            VOTING = 300,
            MISSION = 300
        },
        ["STANDARD"] = {
            NOMINATION = 60,
            VOTING = 30,
            MISSION = 30
        },
        ["COMPETITIVE"] = {
            NOMINATION = 30,
            VOTING = 15,
            MISSION = 15
        }
    }

    local durations = pace_durations[pace] or pace_durations["STANDARD"]
    local seconds = durations[phase] or 30

    local TIME = redis.call("TIME")
    return tonumber(TIME[1]) + seconds
end

redis.register_function("room_start", function(keys, args)
    local KEY_ROOM = keys[1]
    local KEY_ROOM_EVENTS = keys[2]
    local KEY_ROOM_USERS = keys[3]
    local KEY_GAME_SPY = keys[4]
    local KEY_GAME_ORDER = keys[5]
    local KEY_GAME_STATE = keys[6]
    local KEY_MISSIONS = keys[7]

    local OWNER_ID = args[1]
    local EVENT_ID = args[2]

    local room_data = redis.call("HMGET", KEY_ROOM, "owner", "status", "game_pace")
    local current_owner = room_data[1]
    local current_status = room_data[2]
    local game_pace = room_data[3]

    if not current_owner then
        return {"ERR", "ROOM_NOT_FOUND"}
    end
    if current_owner ~= OWNER_ID then
        return {"ERR", "NOT_ROOM_OWNER"}
    end
    if current_status == "PLAYING" then
        return {"ERR", "ROOM_ALREADY_STARTED"}
    end

    local members = redis.call("HKEYS", KEY_ROOM_USERS)
    local member_count = #members

    -- todo remove the second statement, was used for debugging
    if member_count < 5 and member_count ~= 2 then
        return {"ERR", "NOT_ENOUGH_PLAYERS"}
    end

    -- 1. Determine Spy Count
    local spy_counts = {
        [2] = 1,
        [5] = 2,
        [6] = 2,
        [7] = 3,
        [8] = 3,
        [9] = 4,
        [10] = 4
    }
    local num_spies = spy_counts[member_count] or 2

    -- 2. Shuffle Turn Order and Roles Mapping
    local TIME = redis.call("TIME")
    math.randomseed(tonumber(TIME[1]) + tonumber(TIME[2]))

    local order_list = shuffle(copy_table(members))

    -- Generate a list of roles the same size as players
    local roles = {}
    for i = 1, num_spies do
        table.insert(roles, "SPY")
    end
    for i = 1, (member_count - num_spies) do
        table.insert(roles, "AGENT")
    end
    local shuffled_roles = shuffle(roles)

    -- 3. Identify Spy IDs and Save to SET
    local spy_ids = {}
    for i, role in ipairs(shuffled_roles) do
        if role == "SPY" then
            table.insert(spy_ids, order_list[i])
        end
    end

    redis.call("DEL", KEY_GAME_SPY)
    redis.call("SADD", KEY_GAME_SPY, unpack(spy_ids))

    redis.call("DEL", KEY_MISSIONS)

    -- 4. Save Turn Order
    redis.call("DEL", KEY_GAME_ORDER)
    redis.call("RPUSH", KEY_GAME_ORDER, unpack(order_list))

    local expires_at = get_phase_expiration(game_pace, "NOMINATION")

    -- 6. Initialize Game State
    redis.call("DEL", KEY_GAME_STATE)
    redis.call("HSET", KEY_GAME_STATE, "phase", "NOMINATION", "phase_expires_at", expires_at, "event_id", EVENT_ID,
        "member_count", member_count, "round", 1, "failed_missions", 0, "leader_idx", 0, "hammer_idx", member_count - 1)

    -- 7. Update Room Status
    redis.call("HSET", KEY_ROOM, "status", "PLAYING")
    redis.call("HDEL", KEY_ROOM, "result")

    -- 8. Notify Frontend
    redis.call("PUBLISH", KEY_ROOM_EVENTS, "GAME_START")

    return {"OK", "NO_CONTENT", expires_at}
end)

redis.register_function("game_nominate", function(keys, args)
    local KEY_GAME_STATE = keys[1]
    local KEY_ROOM_MEMBERS = keys[2]
    local KEY_MEMBERS_NOMINATED = keys[3]
    local KEY_ROOM_EVENTS = keys[4]
    local KEY_ROOM = keys[5]
    local KEY_ROOM_ORDER = keys[6]
    local KEY_MEMBERS_VOTING = keys[7]

    local LEADER_ID = args[1]
    local NOMINEES_JSON = args[2] -- Pass as JSON string from Go
    local EVENT_ID = args[3]

    -- 1. Existence & Phase Check
    local game_state = redis.call("HMGET", KEY_GAME_STATE, "phase", "leader_idx", "round")
    if not game_state[1] then
        return {"ERR", "GAME_NOT_FOUND"}
    end

    local phase = game_state[1]
    local leader_index = tonumber(game_state[2])
    local round = tonumber(game_state[3])

    if phase ~= "NOMINATION" then
        return {"ERR", "WRONG_PHASE"}
    end

    local leader_id = redis.call("LINDEX", KEY_ROOM_ORDER, leader_index)

    if leader_id ~= LEADER_ID then
        return {"ERR", "NOT_LEADER"}
    end

    -- 2. Determine Required Team Size
    local member_ids = redis.call("HKEYS", KEY_ROOM_MEMBERS)
    local player_count = #member_ids

    -- Matrix: [player_count][round]
    local team_sizes = {
        [2] = {1, 2, 1, 2, 2},
        [5] = {2, 3, 2, 3, 3},
        [6] = {2, 3, 4, 3, 4},
        [7] = {2, 3, 3, 4, 4},
        [8] = {3, 4, 4, 5, 5},
        [9] = {3, 4, 4, 5, 5},
        [10] = {3, 4, 4, 5, 5}
    }

    local required_size = team_sizes[player_count][round]
    local nominees = cjson.decode(NOMINEES_JSON)
    if #nominees ~= required_size then
        return {"ERR", "INVALID_TEAM_SIZE"}
    end

    -- 3. Validate Nominees (Uniqueness and Membership)
    local seen = {}
    for _, id in ipairs(nominees) do
        if seen[id] then
            return {"ERR", "DUPLICATE_NOMINEE"}
        end
        seen[id] = true

        -- Check if nominee is actually in the room
        if redis.call("HEXISTS", KEY_ROOM_MEMBERS, id) == 0 then
            return {"ERR", "NOMINEE_NOT_IN_ROOM"}
        end
    end

    -- remove votes from previous nomination
    redis.call("DEL", KEY_MEMBERS_VOTING)
    -- 4. Execute Nomination
    redis.call("DEL", KEY_MEMBERS_NOMINATED)
    redis.call("SADD", KEY_MEMBERS_NOMINATED, unpack(nominees))

    -- 5. Transition Phase to VOTING
    local game_pace = redis.call("HGET", KEY_ROOM, "game_pace")

    local expires_at = get_phase_expiration(game_pace, "VOTING")

    redis.call("HSET", KEY_GAME_STATE, "phase", "VOTING", "phase_expires_at", expires_at, "event_id", EVENT_ID)

    redis.call("PUBLISH", KEY_ROOM_EVENTS, "GAME_NOMINATION_COMPLETE")

    return {"OK", "NO_CONTENT", expires_at}
end)

redis.register_function("game_nominate_timeout", function(keys, args)
    local KEY_GAME_STATE = keys[1]
    local KEY_ROOM_EVENTS = keys[2]
    local KEY_ROOM = keys[3]
    local KEY_ROOM_ORDER = keys[4]

    local CURRENT_EVENT_ID = args[1]
    local NEW_EVENT_ID = args[2]

    -- Event ID check
    local game_state = redis.call("HMGET", KEY_GAME_STATE, "event_id", "leader_idx", "hammer_idx")
    local game_event_id = game_state[1]
    local leader_idx = tonumber(game_state[2])
    local hammer_idx = tonumber(game_state[3])

    if not game_event_id then
        return {"ERR", "GAME_NOT_FOUND"}
    end

    if CURRENT_EVENT_ID ~= game_event_id then
        return {"ERR", "EVENT_EXPIRED"}
    end

    -- Check if leader had hammer
    if leader_idx == hammer_idx then
        -- todo game end function
        redis.call("HSET", KEY_ROOM, "status", "LOBBY", "result", "HAMMER")

        redis.call("PUBLISH", KEY_ROOM_EVENTS, "GAME_END_HAMMER")
        return {"ERR", "GAME_END"}
    end

    -- Pass leader to next member
    local member_count = redis.call("HGET", KEY_GAME_STATE, "member_count")
    local next_leader_idx = (leader_idx + 1) % member_count
    redis.call("HSET", KEY_GAME_STATE, "leader_idx", next_leader_idx)

    -- Reset timer

    local game_pace = redis.call("HGET", KEY_ROOM, "game_pace")
    local expires_at = get_phase_expiration(game_pace, "NOMINATION")

    redis.call("HSET", KEY_GAME_STATE, "phase_expires_at", expires_at, "event_id", NEW_EVENT_ID)

    redis.call("PUBLISH", KEY_ROOM_EVENTS, "GAME_NOMINATION_NEXT_LEADER")

    return {"OK", "NO_CONTENT", expires_at}
end)

-- Calculate votes
local function calculate_game_votes(KEY_MEMBERS_VOTING, KEY_GAME_STATE, KEY_ROOM_EVENTS, KEY_ROOM,
    KEY_MEMBERS_NOMINATED, player_count, leader_idx, game_pace, hammer_idx, NEXT_TIMEOUT_EVENT_ID)

    local votes = redis.call("HVALS", KEY_MEMBERS_VOTING)
    local approves = 0
    for _, v in ipairs(votes) do
        if v == "1" then
            approves = approves + 1
        end
    end

    local is_approved = approves > (player_count / 2)
    local next_leader = (leader_idx + 1) % player_count

    if is_approved then -- Team was approved 
        -- pass leader and change phase to mission
        local new_hammer = (next_leader + 4) % player_count
        local expires_at = get_phase_expiration(game_pace, "MISSION")
        redis.call("HSET", KEY_GAME_STATE, "phase", "MISSION", "leader_idx", next_leader, "hammer_idx", new_hammer,
            "phase_expires_at", expires_at, "event_id", NEXT_TIMEOUT_EVENT_ID)
        redis.call("PUBLISH", KEY_ROOM_EVENTS, "TEAM_APPROVED")
        return {"OK", "TEAM_APPROVED", expires_at}

    else -- Team was rejected
        if leader_idx == hammer_idx then -- Hammer
            -- reset to lobby and set game result
            redis.call("HSET", KEY_ROOM, "status", "LOBBY", "result", "HAMMER")
            redis.call("DEL", KEY_MEMBERS_VOTING)
            redis.call("DEL", KEY_MEMBERS_NOMINATED)
            redis.call("PUBLISH", KEY_ROOM_EVENTS, "GAME_END_HAMMER")
            return {"OK", "HAMMER", 0}

        else -- No hammer
            local expires_at = get_phase_expiration(game_pace, "NOMINATION")
            -- pass leader and move back to nomination phase
            redis.call("HSET", KEY_GAME_STATE, "phase", "NOMINATION", "leader_idx", next_leader, "phase_expires_at",
                expires_at, "event_id", NEXT_TIMEOUT_EVENT_ID)
            -- remove nominated team so next leader can set them
            redis.call("DEL", KEY_MEMBERS_NOMINATED)

            redis.call("PUBLISH", KEY_ROOM_EVENTS, "GAME_VOTING_REJECTED")
            return {"OK", "TEAM_REJECTED", expires_at}

        end
    end
end

redis.register_function("game_vote", function(keys, args)
    local KEY_GAME_STATE = keys[1]
    local KEY_MEMBERS_VOTING = keys[2]
    local KEY_ROOM_MEMBERS = keys[3]
    local KEY_ROOM_EVENTS = keys[4]
    local KEY_ROOM = keys[5]
    local KEY_MEMBERS_NOMINATED = keys[6]

    local USER_ID = args[1]
    local VOTE = args[2]
    local NEXT_TIMEOUT_EVENT_ID = args[3]

    if VOTE ~= "0" and VOTE ~= "1" then
        return {"ERR", "INVALID_VOTE", 0}
    end

    if redis.call("HEXISTS", KEY_ROOM_MEMBERS, USER_ID) == 0 then
        return {"ERR", "NOT_IN_THIS_ROOM", 0}
    end

    local game_state = redis.call("HMGET", KEY_GAME_STATE, "phase", "leader_idx", "hammer_idx", "member_count")
    local game_phase = game_state[1]
    local leader_idx = tonumber(game_state[2])
    local hammer_idx = tonumber(game_state[3])
    local member_count = tonumber(game_state[4])

    if game_phase ~= "VOTING" then
        return {"ERR", "WRONG_PHASE", 0}
    end

    if redis.call("HEXISTS", KEY_MEMBERS_VOTING, USER_ID) == 1 then
        return {"ERR", "ALREADY_VOTED", 0}
    end

    -- 2. Register Vote
    redis.call("HSET", KEY_MEMBERS_VOTING, USER_ID, VOTE)

    -- 3. Check if all members have voted
    local vote_count = redis.call("HLEN", KEY_MEMBERS_VOTING)

    if vote_count < member_count then
        return {"OK", "NO_CONTENT", 0}
    end

    local game_pace = redis.call("HGET", KEY_ROOM, "game_pace")
    return calculate_game_votes(KEY_MEMBERS_VOTING, KEY_GAME_STATE, KEY_ROOM_EVENTS, KEY_ROOM, KEY_MEMBERS_NOMINATED,
        member_count, leader_idx, game_pace, hammer_idx, NEXT_TIMEOUT_EVENT_ID)
end)

redis.register_function("game_vote_timeout", function(keys, args)
    local KEY_GAME_STATE = keys[1]
    local KEY_MEMBERS_VOTES = keys[2]
    local KEY_ROOM_MEMBERS = keys[3]
    local KEY_ROOM_EVENTS = keys[4]
    local KEY_ROOM = keys[5]
    local KEY_MEMBERS_NOMINATED = keys[6]

    local TIMEOUT_EVENT_ID = args[1]
    local NEXT_TIMEOUT_EVENT_ID = args[2]

    -- 1. State & Event ID Check

    local game_state = redis.call("HMGET", KEY_GAME_STATE, "phase", "event_id", "leader_idx", "hammer_idx")
    local game_phase = game_state[1]
    local event_id = game_state[2]
    local leader_idx = tonumber(game_state[3])
    local hammer_idx = tonumber(game_state[4])
    local game_pace = redis.call("HGET", KEY_ROOM, "game_pace")

    if game_phase ~= "VOTING" then
        return {"OK", "STALE", 0}
    end

    if event_id ~= TIMEOUT_EVENT_ID then
        return {"OK", "STALE", 0}
    end

    -- 2. Fill missing votes with "0" (Reject)
    local all_members = redis.call("HKEYS", KEY_ROOM_MEMBERS)
    local player_count = #all_members
    for _, user_id in ipairs(all_members) do
        if redis.call("HEXISTS", KEY_MEMBERS_VOTES, user_id) == 0 then
            redis.call("HSET", KEY_MEMBERS_VOTES, user_id, "0")
        end
    end

    return calculate_game_votes(KEY_MEMBERS_VOTES, KEY_GAME_STATE, KEY_ROOM_EVENTS, KEY_ROOM, KEY_MEMBERS_NOMINATED,
        player_count, leader_idx, game_pace, hammer_idx, NEXT_TIMEOUT_EVENT_ID)
end)

local function calculate_mission_results(KEY_MEMBERS_VOTING, KEY_MISSION_VOTES, KEY_GAME_STATE, KEY_ROOM_EVENTS,
    KEY_ROOM, KEY_MEMBERS_NOMINATED, KEY_MISSION_RESULTS, player_count, current_round, mission_fails, game_pace,
    NEXT_TIMEOUT_EVENT_ID)

    local votes = redis.call("HVALS", KEY_MISSION_VOTES)
    redis.call("DEL", KEY_MISSION_VOTES)
    local fails = 0
    for _, v in ipairs(votes) do
        if v == "0" then
            fails = fails + 1
        end
    end

    -- Rule: 7+ players, Round 4 requires 2 fails
    local required_fails = 1
    if player_count >= 7 and current_round == 4 then
        required_fails = 2
    end

    local success = fails < required_fails

    local votes_map = redis.call("HGETALL", KEY_MEMBERS_VOTING)
    redis.call("DEL", KEY_MEMBERS_VOTING)
    local mission_members = redis.call("SMEMBERS", KEY_MEMBERS_NOMINATED)
    redis.call("DEL", KEY_MEMBERS_NOMINATED)

    local approved_players = {} -- List of IDs who voted "1"

    -- Parse the HGETALL array
    for i = 1, #votes_map, 2 do
        local uid = votes_map[i]
        local v = votes_map[i + 1]
        if v == "1" then
            table.insert(approved_players, uid)
        end
    end

    -- Archive result
    local result_entry = cjson.encode({
        round = current_round,
        fail_count = fails,
        is_success = success,
        members = mission_members,
        voters = approved_players
    })
    redis.call("RPUSH", KEY_MISSION_RESULTS, result_entry)

    -- Check Win Conditions (Single Source of Truth)
    local total_wins = 0
    local total_fails = 0
    local all_missions = redis.call("LRANGE", KEY_MISSION_RESULTS, 0, -1)

    for _, mission_json in ipairs(all_missions) do
        local mission = cjson.decode(mission_json)
        if mission.is_success then
            total_wins = total_wins + 1
        else
            total_fails = total_fails + 1
        end
    end

    if total_wins >= 3 then
        redis.call("HSET", KEY_ROOM, "status", "LOBBY", "result", "AGENT")
        redis.call("PUBLISH", KEY_ROOM_EVENTS, "GAME_END_AGENT")
        return {"OK", "GAME_END_AGENT", 0}
    elseif total_fails >= 3 then
        redis.call("HSET", KEY_ROOM, "status", "LOBBY", "result", "SPY")
        redis.call("PUBLISH", KEY_ROOM_EVENTS, "GAME_END_SPY")
        return {"OK", "GAME_END_SPY", 0}
    else
        -- Next Round
        local expires_at = get_phase_expiration(game_pace, "NOMINATION")

        redis.call("HMSET", KEY_GAME_STATE, "phase", "NOMINATION", "round", current_round + 1, "failed_missions",
            total_fails, "phase_expires_at", expires_at, "event_id", NEXT_TIMEOUT_EVENT_ID)

        redis.call("PUBLISH", KEY_ROOM_EVENTS, "GAME_MISSION_DONE")
        return {"OK", "GAME_MISSION_DONE", expires_at}
    end
end

redis.register_function("game_mission", function(keys, args)
    local KEY_GAME_STATE = keys[1]
    local KEY_MISSION_VOTES = keys[2]
    local KEY_ROOM_MEMBERS = keys[3]
    local KEY_ROOM_EVENTS = keys[4]
    local KEY_ROOM = keys[5]
    local KEY_MEMBERS_NOMINATED = keys[6]
    local KEY_MISSION_RESULTS = keys[7]
    local KEY_GAME_SPY = keys[8]
    local KEY_MEMBERS_VOTING = keys[9]

    local USER_ID = args[1]
    local VOTE = args[2] -- "1" for Pass, "0" for Fail
    local NEXT_TIMEOUT_EVENT_ID = args[3]

    if VOTE ~= "0" and VOTE ~= "1" then
        return {"ERR", "INVALID_VOTE", 0}
    end

    -- 1. State & Membership Checks
    local game_state = redis.call("HMGET", KEY_GAME_STATE, "phase", "member_count", "round", "failed_missions")
    local phase = game_state[1]
    local member_count = tonumber(game_state[2])
    local round = tonumber(game_state[3])
    local mission_fails = tonumber(game_state[4])
    local game_pace = redis.call("HGET", KEY_ROOM, "game_pace")

    if phase ~= "MISSION" then
        return {"ERR", "WRONG_PHASE", 0}
    end

    if redis.call("SISMEMBER", KEY_MEMBERS_NOMINATED, USER_ID) == 0 then
        return {"ERR", "NOT_NOMINATED", 0}
    end

    if redis.call("HEXISTS", KEY_MISSION_VOTES, USER_ID) == 1 then
        return {"ERR", "ALREADY_VOTED", 0}
    end

    -- 2. Role Check: Only spies can fail
    if VOTE == "0" and redis.call("SISMEMBER", KEY_GAME_SPY, USER_ID) == 0 then
        return {"ERR", "AGENTS_CANNOT_FAIL", 0}
    end

    -- 3. Register Secret Vote
    redis.call("HSET", KEY_MISSION_VOTES, USER_ID, VOTE)

    -- 4. Check if all nominees voted
    local required_votes = redis.call("SCARD", KEY_MEMBERS_NOMINATED)
    local vote_count = redis.call("HLEN", KEY_MISSION_VOTES)

    if vote_count < required_votes then
        return {"OK", "NO_CONTENT", 0}
    end

    return calculate_mission_results(KEY_MEMBERS_VOTING, KEY_MISSION_VOTES, KEY_GAME_STATE, KEY_ROOM_EVENTS, KEY_ROOM,
        KEY_MEMBERS_NOMINATED, KEY_MISSION_RESULTS, member_count, round, mission_fails, game_pace, NEXT_TIMEOUT_EVENT_ID)
end)

redis.register_function("game_mission_timeout", function(keys, args)
    local KEY_GAME_STATE = keys[1]
    local KEY_MISSION_VOTES = keys[2]
    local KEY_ROOM_MEMBERS = keys[3]
    local KEY_ROOM_EVENTS = keys[4]
    local KEY_ROOM = keys[5]
    local KEY_MEMBERS_NOMINATED = keys[6]
    local KEY_MISSION_RESULTS = keys[7]
    local KEY_MEMBERS_VOTING = keys[8]

    local TIMEOUT_EVENT_ID = args[1]
    local NEXT_TIMEOUT_EVENT_ID = args[2]

    -- 1. State & Event ID Check
    local game_state = redis.call("HMGET", KEY_GAME_STATE, "phase", "event_id", "member_count", "round",
        "failed_missions")
    local game_phase = game_state[1]
    local event_id = game_state[2]
    local member_count = tonumber(game_state[3])
    local current_round = tonumber(game_state[4])
    local mission_fails = tonumber(game_state[5])
    local game_pace = redis.call("HGET", KEY_ROOM, "game_pace")

    if game_phase ~= "MISSION" then
        return {"OK", "STALE", 0}
    end

    if event_id ~= TIMEOUT_EVENT_ID then
        return {"OK", "STALE", 0}
    end

    -- 2. Fill missing votes with "1" (Pass)
    -- We only iterate over the nominated members
    local nominated_members = redis.call("SMEMBERS", KEY_MEMBERS_NOMINATED)
    for _, user_id in ipairs(nominated_members) do
        if redis.call("HEXISTS", KEY_MISSION_VOTES, user_id) == 0 then
            -- System forces a PASS ("1") for AFK players
            redis.call("HSET", KEY_MISSION_VOTES, user_id, "1")
        end
    end

    -- 3. Calculate Result using the same logic as the manual vote
    return calculate_mission_results(KEY_MEMBERS_VOTING, KEY_MISSION_VOTES, KEY_GAME_STATE, KEY_ROOM_EVENTS, KEY_ROOM,
        KEY_MEMBERS_NOMINATED, KEY_MISSION_RESULTS, member_count, current_round, mission_fails, game_pace,
        NEXT_TIMEOUT_EVENT_ID)
end)
