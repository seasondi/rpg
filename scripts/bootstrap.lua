rpg.required_mod = {}

_G.system_require = _G.require
_G.require = function(name)
    -- todo: 需要热更的文件收集逻辑调整
    local r =  _G.system_require(name)
    if string.find(name, "gm.") == 1 or string.find(name, "tables.") == 1 and name ~= "init" then
        rpg.required_mod[name] = true
    end
    return r
end

local platform = rpg.platform()

if platform == "windows" then
    package.path = package.path .. ";.\\scripts\\plugin\\?.lua;"
else
    package.path = package.path .. ";./scripts/plugin/?.lua;"
end
require("tablePlus")

local log = require("logger")
local config = require("config")
local stp = require("stackTracePlus")

--__G__TRACEBACK__ = stp.stacktrace
debug.traceback = stp.stacktrace



-- 输出msg的详细信息,该函数会将table展开输出
function detail(msg)
    local cache={}
    local print_str = ""
    local function print_simple(tb)
        return tb.__type == "entity" or tb.__mbType ~= nil
    end
    local function sub_print_r(t, indent)
        if cache[tostring(t)] then
            print_str = print_str .. indent .. "*" .. tostring(t) .. "\n"
        else
            cache[tostring(t)] = true
            if type(t) == "table" then
                if t.__type == "entity" then
                    print_str = print_str .. tostring(t) .. "\n"
                else
                    for pos, val in pairs(t) do
                        if type(val) == "table" then
                            local show_simple = print_simple(val)
                            print_str = print_str .. indent .. "[" .. pos .. "] => "
                            if show_simple == false then
                                print_str = print_str .. tostring(t) .. " {" .. "\n"
                            end
                            sub_print_r(val,indent .. string.rep(" ",string.len(pos) + 8))
                            if show_simple == false then
                                print_str = print_str .. indent..string.rep(" ",string.len(pos) + 6) .. "}" .. "\n"
                            end
                        elseif type(val) == "string" then
                            print_str = print_str .. indent .. "[" .. pos .. '] => "' .. val .. '"' .. "\n"
                        else
                            print_str = print_str .. indent .. "[" .. pos .. "] => " .. tostring(val) .. "\n"
                        end
                        if #print_str >= 1024 then
                            print_str = string.sub(print_str, 0, 1024) .. "\n" .. indent .. "...\n"
                            break
                        end
                    end
                end
            else
                print_str = print_str .. indent .. tostring(t) .. "\n"
            end
        end
    end

    if type(msg) == "table" then
        if print_simple(msg) then
            print_str = print_str .. tostring(msg)
        else
            print_str = print_str .. tostring(msg) .. " {" .. "\n"
            sub_print_r(msg,"  ")
            print_str = print_str .. "}" .. "\n"
        end
    else
        print_str = tostring(msg)
    end

    return print_str
end

_G.origin_print = _G.print
-- 供命令行调试输出用, 必须实现
_G.console_print = function(...)
    if _G.console_output == nil then
        return
    end
    local str = ""
    local args = {...}
    for _, arg in pairs(args) do
        if type(arg) == "table" then
            str = str .. detail(arg)
        else
            str = str .. tostring(arg)
        end
    end
    _G.console_output = _G.console_output .. str .. "\n"
end

local release = config.getConfig("release")

-- print输出到debug日志
_G.print = function(...)
    if release == true then
        return
    end

    local str = ""
    local i = 1
    for k, v in pairs(table.pack(...)) do
        local idx = i
        while idx < k do
            str = str .. "nil"
            idx = idx + 1
        end
        i = idx + 1
        if type(v) == "table" then
            if v.__type == "entity" then
                str = str .. "[" .. v.__name .. ":" .. v.id .. "] "
            else
                str = str .. detail(v)
            end
        else
            str = str .. tostring(v)
        end
    end
    log.debug(str)
end

function show_traceback(message)
    message = message or ""
    print(debug.traceback(message, 1))
end

-- key不区分大小写
--print(config.getConfig("snowFlake.workId"))

--[必须项] 路径相对于config的workPath
if rpg.is_game then
    rpg.scriptPath = "./game"
    if platform == "windows" then
        package.path = package.path .. ";.\\scripts\\game\\?.lua"
        package.path = package.path .. ";.\\scripts\\tables\\?.lua"
    else
        package.path = package.path .. ";./scripts/game/?.lua"
        package.path = package.path .. ";./scripts/tables/?.lua"
    end
    require("init")
elseif rpg.is_robot then
    rpg.scriptPath = "./robot"
    if platform == "windows" then
        package.path = package.path .. ";.\\scripts\\robot\\?.lua"
    else
        package.path = package.path .. ";./scripts/robot/?.lua"
    end
    require("init")
end