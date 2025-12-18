-- mTLS + IP Whitelist Authentication Lua Script for Nginx
-- Validates client certificate and IP address for consortium members

local cjson = require "cjson"
package.path = package.path .. ';/etc/nginx/lua/?.lua'
local auth_common = require "auth_common"

local IP_WHITELIST_FILE = "/etc/nginx/conf.d/consortium-ip-whitelist.json"
local AUDIT_LOG_FILE = "/var/log/nginx/audit.log"

local write_audit_log = auth_common.write_audit_log
local check_auth_failure_limit = auth_common.check_auth_failure_limit

local function load_ip_whitelist()
    local file = io.open(IP_WHITELIST_FILE, "r")
    if not file then
        ngx.log(ngx.ERR, "Failed to open IP whitelist file: ", IP_WHITELIST_FILE)
        return nil
    end

    local content = file:read("*all")
    file:close()

    local ok, whitelist = pcall(cjson.decode, content)
    if not ok then
        ngx.log(ngx.ERR, "Failed to parse IP whitelist file: ", whitelist)
        return nil
    end

    return whitelist
end

-- todo support cidr
local function is_ip_whitelisted(ip, whitelist)
    if not whitelist or not whitelist.members then
        return false
    end

    -- Expect exactly matched IPv4 addresses, e.g. "192.168.1.10".
    -- Users should provide explicit IPs instead of range patterns for clarity.
    for member_id, member_config in pairs(whitelist.members) do
        if member_config.ip_whitelist then
            for _, allowed_ip in ipairs(member_config.ip_whitelist) do
                if ip == allowed_ip then
                    return true, member_id
                end
            end
        end
    end

    return false, nil
end

-- Main authentication function
local function authenticate()
    local verify_result = ngx.var.ssl_client_verify

    -- Get client certificate DN and serial number for audit logging
    local client_cert_dn = ngx.var.ssl_client_s_dn or "-"
    local client_cert_serial = ngx.var.ssl_client_serial or "-"

    local client_ip = ngx.var.remote_addr
    if not client_ip or client_ip == "" then
        client_ip = ngx.var.http_x_forwarded_for
        if client_ip then
            client_ip = client_ip:match("([^,]+)") or client_ip
        end
    end

    if verify_result ~= "SUCCESS" then
        -- Check rate limit for authentication failures
        local rate_limit_key = "mtls_cert_fail:" .. (client_ip or "unknown")
        local is_allowed, rate_limit_err = check_auth_failure_limit(rate_limit_key)
        if not is_allowed then
            ngx.log(ngx.WARN, "[AUTH_FAIL] Rate limit exceeded for certificate verification failures: ip=", client_ip)
            
            local audit_msg = string.format("%s|%s|%s|429|-|-|mTLS|Rate limit exceeded: %s|%s|%s",
                client_ip or "-",
                ngx.var.request_method or "-",
                ngx.var.request_uri or "-",
                rate_limit_err or "Too many failures",
                client_cert_dn,
                client_cert_serial)
            write_audit_log(AUDIT_LOG_FILE, "FAIL", audit_msg)
            
            ngx.status = 429
            ngx.header["Content-Type"] = "application/json"
            ngx.say(cjson.encode({
                error = "Too Many Requests",
                message = rate_limit_err or "Too many authentication failures. Please try again later."
            }))
            ngx.exit(429)
        end
        
        -- Log to error.log (for monitoring/alerting)
        ngx.log(ngx.WARN, "[AUTH_FAIL] Client certificate verification failed: result=", 
                (verify_result or "unknown"),
                ", ip=", client_ip,
                ", method=", ngx.var.request_method,
                ", uri=", ngx.var.request_uri,
                ", cert_dn=", client_cert_dn,
                ", cert_serial=", client_cert_serial)
        
        -- Write to audit.log
        local audit_msg = string.format("%s|%s|%s|401|-|-|mTLS|Certificate verification failed: %s|%s|%s",
            client_ip or "-",
            ngx.var.request_method or "-",
            ngx.var.request_uri or "-",
            verify_result or "unknown",
            client_cert_dn,
            client_cert_serial)
        write_audit_log(AUDIT_LOG_FILE, "FAIL", audit_msg)
        
        ngx.status = 401
        ngx.header["Content-Type"] = "application/json"
        ngx.say(cjson.encode({
            error = "Unauthorized",
            message = "Valid client certificate is required"
        }))
        ngx.exit(401)
    end
    
    -- Verify IP whitelist
    local is_whitelisted, member_id = is_ip_whitelisted(client_ip, load_ip_whitelist())
    if not is_whitelisted then
        -- Check rate limit for authentication failures
        local rate_limit_key = "mtls_ip_fail:" .. (client_ip or "unknown")
        local is_allowed, rate_limit_err = check_auth_failure_limit(rate_limit_key)
        if not is_allowed then
            ngx.log(ngx.WARN, "[AUTH_FAIL] Rate limit exceeded for IP whitelist failures: ip=", client_ip)
            
            local audit_msg = string.format("%s|%s|%s|429|-|-|mTLS|Rate limit exceeded: %s|%s|%s",
                client_ip or "-",
                ngx.var.request_method or "-",
                ngx.var.request_uri or "-",
                rate_limit_err or "Too many failures",
                client_cert_dn,
                client_cert_serial)
            write_audit_log(AUDIT_LOG_FILE, "FAIL", audit_msg)
            
            ngx.status = 429
            ngx.header["Content-Type"] = "application/json"
            ngx.say(cjson.encode({
                error = "Too Many Requests",
                message = rate_limit_err or "Too many authentication failures. Please try again later."
            }))
            ngx.exit(429)
        end
        
        -- Log to error.log (for monitoring/alerting)
        ngx.log(ngx.WARN, "[AUTH_FAIL] IP address not whitelisted: ip=", client_ip,
                ", method=", ngx.var.request_method,
                ", uri=", ngx.var.request_uri,
                ", cert_dn=", client_cert_dn,
                ", cert_serial=", client_cert_serial)
        
        -- Write to audit.log
        local audit_msg = string.format("%s|%s|%s|403|-|-|mTLS|IP address not whitelisted: %s|%s|%s",
            client_ip or "-",
            ngx.var.request_method or "-",
            ngx.var.request_uri or "-",
            client_ip or "-",
            client_cert_dn,
            client_cert_serial)
        write_audit_log(AUDIT_LOG_FILE, "FAIL", audit_msg)
        
        ngx.status = 403
        ngx.header["Content-Type"] = "application/json"
        ngx.say(cjson.encode({
            error = "Forbidden",
            message = "IP address is not whitelisted for consortium member access"
        }))
        ngx.exit(403)
    end
    
    -- Set headers for backend services
    ngx.req.set_header("X-Cert-Subject", client_cert_dn or "-")
    ngx.req.set_header("X-Member-ID", member_id or "-")
    ngx.req.set_header("X-Auth-Method", "mtls")
    
    -- Write successful authentication to audit.log only (not error.log)
    local audit_msg = string.format("%s|%s|%s|200|-|%s|mTLS|SUCCESS|%s|%s",
        client_ip or "-",
        ngx.var.request_method or "-",
        ngx.var.request_uri or "-",
        member_id or "-",
        client_cert_dn,
        client_cert_serial)
    write_audit_log(AUDIT_LOG_FILE, "SUCCESS", audit_msg)
end

-- Execute authentication
authenticate()

