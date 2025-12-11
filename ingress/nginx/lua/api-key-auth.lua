-- API Key Authentication Lua Script for Nginx
-- Validates API Key and passes client identity to backend services

local http = require "resty.http"
local cjson = require "cjson"
package.path = package.path .. ';/etc/nginx/lua/?.lua'
local auth_common = require "auth_common"

-- Configuration: API Key validation method
-- Option 1: File-based lookup (simple, for development/testing)
-- Option 2: Redis lookup (for production)
-- Option 3: External auth service (for production with centralized management)

local AUTH_METHOD = os.getenv("API_KEY_AUTH_METHOD") or "file"
local API_KEYS_FILE = "/etc/nginx/conf.d/api-keys.json"
local REDIS_HOST = os.getenv("REDIS_HOST") or "redis"
local REDIS_PORT = tonumber(os.getenv("REDIS_PORT") or 6379)
local AUTH_SERVICE_URL = os.getenv("AUTH_SERVICE_URL") or "http://auth-service:8080/validate"
local AUDIT_LOG_FILE = "/var/log/nginx/audit.log"

local validate_activation_window = auth_common.validate_activation_window
local ensure_permission = auth_common.ensure_permission
local write_audit_log_common = auth_common.write_audit_log

local function write_audit_log(level, message)
    return write_audit_log_common(AUDIT_LOG_FILE, level, message)
end

local function get_required_permission()
    local method = ngx.var.request_method or ""
    local uri = ngx.var.uri or ""

    if method == "POST" and uri == "/v1/logs" then
        return "submit_log"
    elseif method == "GET" and uri:find("^/status/") then
        return "query_status"
    elseif method == "POST" and uri == "/query_by_content" then
        return "query_by_content"
    else
        return nil
    end
end

-- Read API keys from file (for file-based auth)
local function load_api_keys_from_file()
    local file = io.open(API_KEYS_FILE, "r")
    if not file then
        ngx.log(ngx.ERR, "Failed to open API keys file: ", API_KEYS_FILE)
        return nil
    end

    local content = file:read("*all")
    file:close()

    local ok, api_keys = pcall(cjson.decode, content)
    if not ok then
        ngx.log(ngx.ERR, "Failed to parse API keys file: ", api_keys)
        return nil
    end

    return api_keys
end

-- Validate API Key from file
local function validate_api_key_file(api_key)
    local api_keys = load_api_keys_from_file()
    if not api_keys then
        return nil, "Internal authentication error"
    end

    -- Lookup API key
    local client_info = api_keys[api_key]
    if not client_info then
        return nil, "Invalid API key"
    end

    -- Check if key is active
    if client_info.status ~= "active" then
        return nil, "API key is not active"
    end

    return client_info, nil
end

-- Validate API Key from Redis (with connection reuse for better performance)
local function validate_api_key_redis(api_key)
    local redis = require "resty.redis"
    local red = redis:new()

    red:set_timeout(1000)

    -- Connect to Redis and OpenResty will reuse connections via keepalive
    local ok, err = red:connect(REDIS_HOST, REDIS_PORT)
    if not ok then
        ngx.log(ngx.ERR, "Failed to connect to Redis: ", err)
        return nil, "Authentication service unavailable"
    end

    -- Get client info from Redis
    local client_info_json, err = red:get("api_key:" .. api_key)

    -- Check for operation errors
    if err or not client_info_json or client_info_json == ngx.null then
        ngx.log(ngx.ERR, "Redis GET error: ", err)
        redis:close()
        return nil, "Authentication service error"
    end

    -- Check if key exists
    if not client_info_json or client_info_json == ngx.null then
        -- Return connection to keepalive pool for reuse
        local ok, err = red:set_keepalive(10000, 100)
        if not ok then
            ngx.log(ngx.ERR, "Failed to set keepalive: ", err)
            red:close()
        end
        return nil, "Invalid API key"
    end

    -- Return connection to keepalive pool for reuse (instead of closing)
    -- This allows the connection to be reused by subsequent requests, improving performance
    -- 10s idle timeout, max 100 connections in pool
    local ok, err = red:set_keepalive(10000, 100)
    if not ok then
        ngx.log(ngx.ERR, "Failed to set keepalive: ", err)
        red:close()  -- Fallback to close if keepalive fails
    end

    -- Parse JSON
    local ok, client_info = pcall(cjson.decode, client_info_json)
    if not ok then
        ngx.log(ngx.ERR, "Failed to parse client info from Redis: ", client_info)
        return nil, "Authentication service error"
    end

    if client_info.status ~= "active" then
        return nil, "API key is not active"
    end

    return client_info, nil
end

-- Validate API Key via external auth service
local function validate_api_key_service(api_key)
    local httpc = http.new()
    httpc:set_timeout(1000) -- 1 second timeout

    local res, err = httpc:request_uri(AUTH_SERVICE_URL, {
        method = "POST",
        headers = {
            ["Content-Type"] = "application/json",
        },
        body = cjson.encode({
            api_key = api_key
        })
    })

    if not res then
        ngx.log(ngx.ERR, "Failed to call auth service: ", err)
        return nil, "Authentication service unavailable"
    end

    if res.status ~= 200 then
        return nil, "Invalid API key"
    end

    local ok, result = pcall(cjson.decode, res.body)
    if not ok then
        ngx.log(ngx.ERR, "Failed to parse auth service response: ", result)
        return nil, "Authentication service error"
    end

    if not result.valid then
        return nil, result.error or "Invalid API key"
    end

    return result, nil
end

-- Main authentication function
local function authenticate()
    -- Get API Key from header
    local api_key = ngx.var.http_x_api_key

    if not api_key or api_key == "" then
        ngx.log(ngx.WARN, "[AUTH_FAIL] Missing API key: ip=", ngx.var.remote_addr,
                ", method=", ngx.var.request_method,
                ", uri=", ngx.var.request_uri,
                ", user_agent=", (ngx.var.http_user_agent or "N/A"))

        -- Write to audit.log
        local audit_msg = string.format("%s|%s|%s|401|-|-|API_KEY|Missing API key",
            ngx.var.remote_addr or "-",
            ngx.var.request_method or "-",
            ngx.var.request_uri or "-")
        write_audit_log("FAIL", audit_msg)

        ngx.status = 401
        ngx.header["Content-Type"] = "application/json"
        ngx.say(cjson.encode({
            error = "Unauthorized",
            message = "API key is required. Please provide X-API-Key header or api_key query parameter."
        }))
        ngx.exit(401)
    end

    -- Validate API key based on configured method
    local client_info, err
    if AUTH_METHOD == "redis" then
        client_info, err = validate_api_key_redis(api_key)
    elseif AUTH_METHOD == "service" then
        client_info, err = validate_api_key_service(api_key)
    else
        -- Default to file-based
        client_info, err = validate_api_key_file(api_key)
    end

    if not client_info then
        -- Mask API key for security
        local masked_key = string.sub(api_key, 1, 8) .. "..." .. string.sub(api_key, -4)

        ngx.log(ngx.WARN, "[AUTH_FAIL] API key validation failed: key=", masked_key,
                ", error=", (err or "unknown error"),
                ", ip=", ngx.var.remote_addr,
                ", method=", ngx.var.request_method,
                ", uri=", ngx.var.request_uri)

        -- Write to audit.log
        local audit_msg = string.format("%s|%s|%s|401|-|-|API_KEY|%s",
            ngx.var.remote_addr or "-",
            ngx.var.request_method or "-",
            ngx.var.request_uri or "-",
            err or "Invalid API key")
        write_audit_log("FAIL", audit_msg)

        ngx.status = 401
        ngx.header["Content-Type"] = "application/json"
        ngx.say(cjson.encode({
            error = "Unauthorized",
            message = err or "Invalid API key"
        }))
        ngx.exit(401)
    end

    -- Ensure activation window is valid
    local activation_ok, activation_err = validate_activation_window(client_info)
    if not activation_ok then
        ngx.log(ngx.WARN, "[AUTH_FAIL] API key timing validation failed: client_id=", client_info.client_id or "unknown",
                ", error=", activation_err,
                ", ip=", ngx.var.remote_addr,
                ", method=", ngx.var.request_method,
                ", uri=", ngx.var.request_uri)

        local audit_msg = string.format("%s|%s|%s|403|%s|%s|API_KEY|%s",
            ngx.var.remote_addr or "-",
            ngx.var.request_method or "-",
            ngx.var.request_uri or "-",
            client_info.client_id or "-",
            client_info.org_id or "-",
            activation_err or "Timing validation failed")
        write_audit_log("FAIL", audit_msg)

        ngx.status = 403
        ngx.header["Content-Type"] = "application/json"
        ngx.say(cjson.encode({
            error = "Forbidden",
            message = activation_err
        }))
        ngx.exit(403)
    end

    -- Ensure the API key grants permission for this endpoint
    local permission_ok, permission_err = ensure_permission(client_info, get_required_permission())
    if not permission_ok then
        ngx.log(ngx.WARN, "[AUTH_FAIL] API key permission validation failed: client_id=", client_info.client_id or "unknown",
                ", error=", permission_err or "permission denied",
                ", ip=", ngx.var.remote_addr,
                ", method=", ngx.var.request_method,
                ", uri=", ngx.var.request_uri)

        local audit_msg = string.format("%s|%s|%s|403|%s|%s|API_KEY|%s",
            ngx.var.remote_addr or "-",
            ngx.var.request_method or "-",
            ngx.var.request_uri or "-",
            client_info.client_id or "-",
            client_info.org_id or "-",
            permission_err or "Permission denied")
        write_audit_log("FAIL", audit_msg)

        ngx.status = 403
        ngx.header["Content-Type"] = "application/json"
        ngx.say(cjson.encode({
            error = "Forbidden",
            message = permission_err or "Permission denied"
        }))
        ngx.exit(403)
    end

    -- Set headers for backend services
    ngx.req.set_header("X-API-Client-ID", client_info.client_id)
    if client_info.org_id then
        ngx.req.set_header("X-Client-Org-ID", client_info.org_id)
    end

    local audit_msg = string.format("%s|%s|%s|200|%s|%s|API_KEY|SUCCESS",
        ngx.var.remote_addr or "-",
        ngx.var.request_method or "-",
        ngx.var.request_uri or "-",
        client_info.client_id or "-",
        client_info.org_id or "-")
    write_audit_log("SUCCESS", audit_msg)
end

authenticate()

