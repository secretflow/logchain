-- gRPC API Key Authentication Lua Script for Nginx
-- Validates API Key from gRPC metadata and passes client identity

-- Reuse the HTTP API key authentication logic
-- gRPC metadata is passed as HTTP headers with "grpc-" prefix

local http = require "resty.http"
local cjson = require "cjson"
package.path = package.path .. ';/etc/nginx/lua/?.lua'
local auth_common = require "auth_common"

-- Configuration
local AUTH_METHOD = os.getenv("API_KEY_AUTH_METHOD") or "file"
local API_KEYS_FILE = "/etc/nginx/conf.d/api-keys.json"
local REDIS_HOST = os.getenv("REDIS_HOST") or "redis"
local REDIS_PORT = tonumber(os.getenv("REDIS_PORT") or 6379)
local AUTH_SERVICE_URL = os.getenv("AUTH_SERVICE_URL") or "http://auth-service:8080/validate"
local AUDIT_LOG_FILE = "/var/log/nginx/audit.log"
local REQUIRED_GRPC_PERMISSION = "submit_log"

local validate_activation_window = auth_common.validate_activation_window
local ensure_permission = auth_common.ensure_permission
local write_audit_log_common = auth_common.write_audit_log
local check_auth_failure_limit = auth_common.check_auth_failure_limit

local function write_audit_log(level, message)
    return write_audit_log_common(AUDIT_LOG_FILE, level, message)
end

-- Load API keys from file
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
    
    local client_info = api_keys[api_key]
    if not client_info then
        return nil, "Invalid API key"
    end
    
    if client_info.status ~= "active" then
        return nil, "API key is not active"
    end
    
    return client_info, nil
end

-- Validate API Key from Redis (with connection reuse for better performance)
local function validate_api_key_redis(api_key)
    local redis = require "resty.redis"
    local red = redis:new()
    
    red:set_timeout(1000) -- 1 second timeout
    
    -- Connect to Redis (OpenResty will reuse connections via keepalive)
    local ok, err = red:connect(REDIS_HOST, REDIS_PORT)
    if not ok then
        ngx.log(ngx.ERR, "Failed to connect to Redis: ", err)
        return nil, "Authentication service unavailable"
    end
    
    -- Get client info from Redis
    local client_info_json, err = red:get("api_key:" .. api_key)
    
    -- Check for operation errors
    if err then
        local ok, err = red:set_keepalive(10000, 100)
        if not ok then
            ngx.log(ngx.ERR, "Failed to set keepalive: ", err)
            red:close()
        end
        ngx.log(ngx.ERR, "Redis GET error: ", err)
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
    -- 10s idle timeout, max 100 connections
    local ok, err = red:set_keepalive(10000, 100)
    if not ok then
        ngx.log(ngx.ERR, "Failed to set keepalive: ", err)
        red:close()  -- Fallback to close if keepalive fails
    end
    
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
    httpc:set_timeout(1000)
    
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
    -- Get client IP for rate limiting
    local client_ip = ngx.var.remote_addr
    if not client_ip or client_ip == "" then
        client_ip = ngx.var.http_x_forwarded_for
        if client_ip then
            client_ip = client_ip:match("([^,]+)") or client_ip
        end
    end
    
    -- Get API Key from gRPC metadata
    -- gRPC metadata is passed as HTTP headers, typically as "x-api-key" or in metadata
    local api_key = ngx.var.http_x_api_key or 
                    ngx.var.http_grpc_metadata_x_api_key or
                    ngx.var.http_grpc_metadata_api_key
    
    if not api_key or api_key == "" then
        -- Check rate limit for missing API key (based on IP)
        local rate_limit_key = "grpc_apikey_missing:" .. (client_ip or "unknown")
        local is_allowed, rate_limit_err = check_auth_failure_limit(rate_limit_key)
        if not is_allowed then
            ngx.log(ngx.WARN, "[AUTH_FAIL] Rate limit exceeded for missing API key in gRPC request: ip=", client_ip)
            
            local audit_msg = string.format("%s|%s|%s|429|-|-|GRPC_API_KEY|Rate limit exceeded: %s",
                client_ip or "-",
                "gRPC",
                ngx.var.request_uri or "-",
                rate_limit_err or "Too many failures")
            write_audit_log("FAIL", audit_msg)
            
            ngx.status = 429
            ngx.header["Content-Type"] = "application/grpc"
            ngx.header["Grpc-Status"] = "8"  -- RESOURCE_EXHAUSTED
            ngx.header["Grpc-Message"] = rate_limit_err or "Too many authentication failures. Please try again later."
            ngx.exit(429)
        end
        
        -- Log to error.log (for monitoring/alerting)
        ngx.log(ngx.WARN, "[AUTH_FAIL] Missing API key in gRPC request: ip=", client_ip,
                ", uri=", ngx.var.request_uri)
        
        -- Write to audit.log
        local audit_msg = string.format("%s|%s|%s|401|-|-|GRPC_API_KEY|Missing API key",
            client_ip or "-",
            "gRPC",
            ngx.var.request_uri or "-")
        write_audit_log("FAIL", audit_msg)
        
        ngx.status = 401
        ngx.header["Content-Type"] = "application/grpc"
        ngx.header["Grpc-Status"] = "16"  -- UNAUTHENTICATED
        ngx.header["Grpc-Message"] = "API key is required"
        ngx.exit(401)
    end
    
    -- Validate API key
    local client_info, err
    if AUTH_METHOD == "redis" then
        client_info, err = validate_api_key_redis(api_key)
    elseif AUTH_METHOD == "service" then
        client_info, err = validate_api_key_service(api_key)
    else
        client_info, err = validate_api_key_file(api_key)
    end
    
    if not client_info then
        -- Check rate limit for authentication failures
        -- Limit by both IP and API key to prevent both IP-based and key-based brute force attacks
        local rate_limit_key_ip = "grpc_apikey_fail_ip:" .. (client_ip or "unknown")
        local rate_limit_key_apikey = "grpc_apikey_fail_key:" .. api_key
        
        -- Check IP-based limit
        local is_allowed_ip, rate_limit_err_ip = check_auth_failure_limit(rate_limit_key_ip)
        if not is_allowed_ip then
            local masked_key = string.sub(api_key, 1, 8) .. "..." .. string.sub(api_key, -4)
            ngx.log(ngx.WARN, "[AUTH_FAIL] Rate limit exceeded (IP-based) for gRPC API key validation: ip=", client_ip, ", key=", masked_key)
            
            local audit_msg = string.format("%s|%s|%s|429|-|-|GRPC_API_KEY|Rate limit exceeded (IP): %s",
                client_ip or "-",
                "gRPC",
                ngx.var.request_uri or "-",
                rate_limit_err_ip or "Too many failures")
            write_audit_log("FAIL", audit_msg)
            
            ngx.status = 429
            ngx.header["Content-Type"] = "application/grpc"
            ngx.header["Grpc-Status"] = "8"  -- RESOURCE_EXHAUSTED
            ngx.header["Grpc-Message"] = rate_limit_err_ip or "Too many authentication failures. Please try again later."
            ngx.exit(429)
        end
        
        -- Check API key-based limit
        local is_allowed_key, rate_limit_err_key = check_auth_failure_limit(rate_limit_key_apikey)
        if not is_allowed_key then
            local masked_key = string.sub(api_key, 1, 8) .. "..." .. string.sub(api_key, -4)
            ngx.log(ngx.WARN, "[AUTH_FAIL] Rate limit exceeded (Key-based) for gRPC API key validation: ip=", client_ip, ", key=", masked_key)
            
            local audit_msg = string.format("%s|%s|%s|429|-|-|GRPC_API_KEY|Rate limit exceeded (Key): %s",
                client_ip or "-",
                "gRPC",
                ngx.var.request_uri or "-",
                rate_limit_err_key or "Too many failures")
            write_audit_log("FAIL", audit_msg)
            
            ngx.status = 429
            ngx.header["Content-Type"] = "application/grpc"
            ngx.header["Grpc-Status"] = "8"  -- RESOURCE_EXHAUSTED
            ngx.header["Grpc-Message"] = rate_limit_err_key or "Too many authentication failures. Please try again later."
            ngx.exit(429)
        end
        
        -- Mask API key for security
        local masked_key = string.sub(api_key, 1, 8) .. "..." .. string.sub(api_key, -4)
        
        -- Log to error.log (for monitoring/alerting)
        ngx.log(ngx.WARN, "[AUTH_FAIL] gRPC API key validation failed: key=", masked_key,
                ", error=", (err or "unknown error"),
                ", ip=", client_ip,
                ", uri=", ngx.var.request_uri)
        
        -- Write to audit.log
        local audit_msg = string.format("%s|%s|%s|401|-|-|GRPC_API_KEY|%s",
            client_ip or "-",
            "gRPC",
            ngx.var.request_uri or "-",
            err or "Invalid API key")
        write_audit_log("FAIL", audit_msg)
        
        ngx.status = 401
        ngx.header["Content-Type"] = "application/grpc"
        ngx.header["Grpc-Status"] = "16"  -- UNAUTHENTICATED
        ngx.header["Grpc-Message"] = err or "Invalid API key"
        ngx.exit(401)
    end
    
    -- Ensure activation window is valid
    local activation_ok, activation_err = validate_activation_window(client_info)
    if not activation_ok then
        ngx.log(ngx.WARN, "[AUTH_FAIL] gRPC API key timing validation failed: client_id=", client_info.client_id or "unknown",
                ", error=", activation_err,
                ", ip=", ngx.var.remote_addr,
                ", uri=", ngx.var.request_uri)

        local audit_msg = string.format("%s|%s|%s|403|%s|%s|GRPC_API_KEY|%s",
            ngx.var.remote_addr or "-",
            "gRPC",
            ngx.var.request_uri or "-",
            client_info.client_id or "-",
            client_info.org_id or "-",
            activation_err)
        write_audit_log("FAIL", audit_msg)

        ngx.status = 403
        ngx.header["Content-Type"] = "application/grpc"
        ngx.header["Grpc-Status"] = "7" -- PERMISSION_DENIED
        ngx.header["Grpc-Message"] = activation_err
        ngx.exit(403)
    end

    -- Ensure permission is granted
    local permission_ok, permission_err = ensure_permission(client_info, REQUIRED_GRPC_PERMISSION)
    if not permission_ok then
        ngx.log(ngx.WARN, "[AUTH_FAIL] gRPC API key permission validation failed: client_id=", client_info.client_id or "unknown",
                ", error=", permission_err or "permission denied",
                ", ip=", ngx.var.remote_addr,
                ", uri=", ngx.var.request_uri)

        local audit_msg = string.format("%s|%s|%s|403|%s|%s|GRPC_API_KEY|%s",
            ngx.var.remote_addr or "-",
            "gRPC",
            ngx.var.request_uri or "-",
            client_info.client_id or "-",
            client_info.org_id or "-",
            permission_err or "Permission denied")
        write_audit_log("FAIL", audit_msg)

        ngx.status = 403
        ngx.header["Content-Type"] = "application/grpc"
        ngx.header["Grpc-Status"] = "7" -- PERMISSION_DENIED
        ngx.header["Grpc-Message"] = permission_err or "Permission denied"
        ngx.exit(403)
    end

    -- Set headers for backend services (gRPC metadata)
    ngx.req.set_header("X-API-Client-ID", client_info.client_id)
    if client_info.org_id then
        ngx.req.set_header("X-Client-Org-ID", client_info.org_id)
    end

    local audit_msg = string.format("%s|%s|%s|200|%s|%s|GRPC_API_KEY|SUCCESS",
        client_ip or "-",
        "gRPC",
        ngx.var.request_uri or "-",
        client_info.client_id or "-",
        client_info.org_id or "-")
    write_audit_log("SUCCESS", audit_msg)
end

authenticate()

