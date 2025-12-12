-- Common helpers for API key and mTLS authentication
-- Shared by api-key-auth.lua and grpc-api-key-auth.lua

local cjson = require "cjson"

local _M = {}

local CLOCK_SKEW_SECONDS = tonumber(os.getenv("API_KEY_CLOCK_SKEW")) or 10 -- default 10 seconds
local cjson_null = cjson.null

local function calculate_timezone_offset()
    local now = os.time()
    local utc_table = os.date("!*t", now)
    utc_table.isdst = false
    local utc_epoch = os.time(utc_table)
    return now - utc_epoch
end

local TZ_OFFSET = calculate_timezone_offset()

local function is_null(value)
    return value == nil or value == cjson_null
end

local function parse_iso8601_utc(value)
    if type(value) ~= "string" then
        return nil, "timestamp is missing"
    end

    local year, month, day, hour, min, sec =
        value:match("^(%d%d%d%d)%-(%d%d)%-(%d%d)T(%d%d):(%d%d):(%d%d)Z$")
    if not year then
        return nil, "timestamp must be in ISO8601 UTC format (YYYY-MM-DDTHH:MM:SSZ)"
    end

    local tbl = {
        year = tonumber(year),
        month = tonumber(month),
        day = tonumber(day),
        hour = tonumber(hour),
        min = tonumber(min),
        sec = tonumber(sec),
        isdst = false,
    }

    local local_epoch = os.time(tbl)
    if not local_epoch then
        return nil, "invalid timestamp value"
    end

    return local_epoch - TZ_OFFSET, nil
end

local function validate_activation_window(client_info)
    if is_null(client_info.created_at) then
        return nil, "API key is missing created_at"
    else
        local created_ts, err = parse_iso8601_utc(client_info.created_at)
        if not created_ts then
            return nil, err or "Invalid created_at value"
        elseif created_ts > ngx.time() + CLOCK_SKEW_SECONDS then
            return nil, "API key is not yet active"
        end
    end

    if is_null(client_info.expires_at) then
        return nil, "API key is missing expires_at"
    else
        local now = ngx.time()
        local expires_ts, exp_err = parse_iso8601_utc(client_info.expires_at)
        if not expires_ts then
            return nil, exp_err or "Invalid expires_at value"
        elseif expires_ts <= now - CLOCK_SKEW_SECONDS then
            return nil, "API key has expired"
        else
            return true
        end
    end
end

local function ensure_permission(client_info, required_permission)
    if not required_permission then
        return nil, "invalid method and uri has no permissions assigned"
    end

    local permissions = client_info.permissions
    if is_null(permissions) then
        return nil, "API key has no permissions assigned"
    end

    if type(permissions) ~= "table" then
        return nil, "API key permissions must be an array"
    end

    for _, perm in ipairs(permissions) do
        if perm == required_permission then
            return true
        end
    end

    return nil, string.format("Permission '%s' is required", required_permission)
end

local function write_audit_log(audit_log_file, level, message)
    local audit_file = io.open(audit_log_file, "a")
    if audit_file then
        local timestamp = ngx.var.time_iso8601
        if not timestamp or timestamp == "" then
            timestamp = os.date("!%Y-%m-%dT%TZ")
        end
        audit_file:write(level, "|", timestamp, "|", message, "\n")
        audit_file:close()
    else
        ngx.log(ngx.ERR, "Failed to open audit log file: ", audit_log_file)
    end
end

-- Rate limiting configuration for authentication failures
-- Default: 5 failures per 60 seconds per IP/Key
local MAX_AUTH_FAILURES = tonumber(os.getenv("MAX_AUTH_FAILURES")) or 5
local AUTH_FAILURE_WINDOW = tonumber(os.getenv("AUTH_FAILURE_WINDOW")) or 60  -- seconds

-- Check and record authentication failure using lua-resty-limit-traffic
-- Returns: is_allowed (boolean), err (string or nil)
local function check_auth_failure_limit(key)
    local limit_count = require "resty.limit.count"
    
    -- Create a limit object with shared memory zone
    -- The shared memory zone should be defined in nginx.conf as: lua_shared_dict auth_failures 10m;
    local lim, err = limit_count.new("auth_failures", MAX_AUTH_FAILURES, AUTH_FAILURE_WINDOW)
    if not lim then
        ngx.log(ngx.ERR, "Failed to create limit object: ", err)
        -- On error, allow the request to proceed (fail open)
        return true, nil
    end
    
    -- Check if the key has exceeded the failure limit
    local delay, err = lim:incoming(key, true)
    if not delay then
        if err == "rejected" then
            -- Limit exceeded
            ngx.log(ngx.WARN, "[RATE_LIMIT] Authentication failure limit exceeded for key: ", key)
            return false, "Too many authentication failures. Please try again later."
        else
            ngx.log(ngx.ERR, "Failed to check limit: ", err)
            -- On error, allow the request to proceed (fail open)
            return true, nil
        end
    end
    
    -- Limit not exceeded
    return true, nil
end

_M.CLOCK_SKEW_SECONDS = CLOCK_SKEW_SECONDS
_M.is_null = is_null
_M.parse_iso8601_utc = parse_iso8601_utc
_M.validate_activation_window = validate_activation_window
_M.ensure_permission = ensure_permission
_M.write_audit_log = write_audit_log
_M.check_auth_failure_limit = check_auth_failure_limit
_M.MAX_AUTH_FAILURES = MAX_AUTH_FAILURES
_M.AUTH_FAILURE_WINDOW = AUTH_FAILURE_WINDOW

return _M


