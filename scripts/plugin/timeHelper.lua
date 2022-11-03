local time_date = require("date")

local helper = {
    time_offset = 0,
}

os.system_time = os.time
os.time = function()
    return os.system_time() + helper.time_offset
end

function helper.get_time_offset(date_str)
    local now = os.time()
    local tm = helper.date2Timestamp(date_str)
    return tm - now
end

function helper.set_server_time(date_str)
    local now = os.time()
    local tm = helper.date2Timestamp(date_str)
    local offset = tm - now
    if offset >= 0 then
        helper.time_offset = offset
    end
    return offset
end

function helper.reset_server_time()
    helper.time_offset = 0
end

function helper.date2Timestamp(date_str)
    return time_date(date_str):timestamp()
end

function helper.timestamp2Date(timestamp, format)
    return os.date(format, timestamp)
end

function helper.now_str(format)
    if format == nil then
        format = "%Y-%m-%d %H:%M:%S"
    end
    return os.date(format, os.time())
end

return helper