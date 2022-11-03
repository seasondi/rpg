require("gm.common")
local timeHelper = require("timeHelper")

GMServer = {}

-- ---------------------------------------------------------------------------------- --
GMServer.set_server_time = NewGMCommand("设置服务器时间", GMAuthorityDebug,
        GMString("time_str", "时间(格式: YYYY-MM-DD HH:MM:SS)")
)
GMServer.set_server_time.callback = function(time_str)
    if time_str == "" then
        return "时间字符串不能为空"
    end
    local offset = timeHelper.get_time_offset(time_str)
    if offset < 0 then
        return "设置服务器时间失败, 时间不能小于当前服务器时间. 当前服务器时间: " .. timeHelper.now_str()
    end

    -- 设置时间偏移
    local ret = rpg.setTimeOffset(offset, true)
    if ret == true then
        return "服务器时间成功, 当前时间为: " .. timeHelper.now_str()
    else
        return "服务器时间失败, 当前时间为: " .. timeHelper.now_str()
    end
end

-- ---------------------------------------------------------------------------------- --
GMServer.get_server_time = NewGMCommand("获取服务器时间", GMAuthorityDebug
)
GMServer.get_server_time.callback = function()
    return "当前时间为: " .. timeHelper.now_str()
end
