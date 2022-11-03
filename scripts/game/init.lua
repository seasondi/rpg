rpg.implementsPath = "./game"

--local platform = rpg.platform()

--rpg.implementsPath = "./game/imp"
--if platform == "windows" then
--    package.path = package.path .. ";.\\scripts\\game\\imp\\?.lua"
--else
--    package.path = package.path .. ";./scripts/game/imp/?.lua"
--end

require("tablePlus")
require("timeHelper")

local log = require("logger")
local config = require("config")
local gm = require("gm.init")

local all_stubs = {
    ["game_1"] = {
        "RoleStub", "GMStub",
    }
}

function create_stubs()
    if rpg.is_stubs_loaded then
        return true
    end
    local server_key = config.getServerKey()
    for key, stubs in pairs(all_stubs) do
        if key == server_key then
            for _, stub_name in ipairs(stubs) do
                id = rpg.createEntityLocally(stub_name)
                if id == 0 then
                    log.error("create stub: ", stub_name, " failed")
                    return false
                else
                    log.info("create stub: ", stub_name, " success, id: ", id)
                end
            end
        end
    end
    rpg.is_stubs_loaded = true
    return true
end

-- 执行gm命令
function rpg.do_gm_command(json_str)
    return gm:do_gm_command(json_str)
end

-- 脚本层初始化
function rpg.init_server()
    log.info("call rpg.init_server")
    local ret = create_stubs()
    if ret == true then
        gm:init()
    end
    return ret
end

--脚本层结束进程
function rpg.stop_server()
    log.info("call rpg.stop_server")
    for _, ent in pairs(rpg.entities) do
        ent:destroy(true)
    end
end

-- 引擎层时间被修改后回调
function rpg.on_server_time_update(date_str)
    print("server time change to: ", date_str)
    local timeHelper = require("timeHelper")
    timeHelper.set_server_time(date_str)
end

-- 热更
function rpg.on_reload()
    log.info("reload start")
    local reload = require("reload")

    local get_script_reload_files = function()
        if type(rpg.required_mod) ~= "table" then
            return {}
        end
        local t = {}
        for name, _ in pairs(rpg.required_mod) do
            table.insert(t, name)
        end
        return t
    end

    local reload_list = table.concat_array(get_script_reload_files(), rpg.getReloadFiles())
    local ret, info = reload.reload(reload_list)
    if ret ~= true then
        log.info("reload fail: " .. info)
    else
        log.info("reload success")
        gm:refresh()
    end
end