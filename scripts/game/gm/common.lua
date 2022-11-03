GMAuthorityDebug = 0

function NewGMCommand(command_name, authority, ...)
    return {
        ["name"] = command_name,
        ["authority"] = authority,
        ["args"] = table.pack(...),
        ["callback"] = function()
            return "未实现GM函数"
        end,
    }
end

function GMEntityId()
    return {
        ["type"] = "number",
        ["index"] = "entity_id",
        ["name"] = "ENTITY_ID",
    }
end

function GMNumber(index, name)
    return {
        ["type"] = "number",
        ["index"] = index,
        ["name"] = name,
    }
end

function GMString(index, name)
    return {
        ["type"] = "string",
        ["index"] = index,
        ["name"] = name,
    }
end

function GMBool(index, name)
    return {
        ["type"] = "bool",
        ["index"] = index,
        ["name"] = name,
    }
end