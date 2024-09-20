local json = require("json")
require("gm.avatar")
require("gm.server")

local gm = {}
gm.all_gms = {}

function gm:init()
    self:refresh()
end

function gm:refresh()
    self.all_gms = {}
    for _, gm_info in ipairs(self:get_gm_list()) do
        for command, info in pairs(gm_info.value) do
            self.all_gms[command] = info
        end
    end
end

function gm:get_gm_list()
    return {
        {
            ["name"] = "角色指令",
            ["value"] = GMAvatar,
        },
        {
            ["name"] = "系统指令",
            ["value"] = GMServer,
        },
    }
end

function gm:do_gm_command(jsonStr)
    local args = json.decode(jsonStr)
    if args.command ~= nil then
        local info = self.all_gms[args.command]
        if info == nil then
            return "指令不存在"
        end
        local arg_list = {}
        for idx, arg in ipairs(info.args) do
            local val = args[arg.index]
            if arg.type == "number" then
                if val == nil then
                    val = 0
                else
                    val = tonumber(val)
                end
            elseif arg.type == "string" then
                if val == nil then
                    val = ""
                else
                    val = tostring(val)
                end
            end
            if arg.index == "entity_id" then
                local ent = rpg.entities[val]
                if ent == nil then
                    return string.format("entity[%s] not found", val)
                end
                arg_list[idx] = ent
            else
                arg_list[idx] = val
            end
        end
        print("call gm ", args.command, ", args: ", arg_list)
        return info.callback(table.unpack(arg_list))
    end
    return "无效的指令: " .. detail(args)
end

return gm