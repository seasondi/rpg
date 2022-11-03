-- table中所有的key
function table.keys(t)
    local keys = {}
    local i = 1
    for key, _ in pairs(t) do
        keys[i] = key
        i = i + 1
    end
    return keys
end

-- table中所有的value
function table.values(t)
    local values = {}
    local i = 1
    for _, value in pairs(t) do
        values[i] = value
        i = i + 1
    end
    return values
end

-- table长度
function table.length(t)
    local length = 0
    for _, __ in pairs(t) do
        length = length + 1
    end
    return length
end

-- table是否为空
function table.empty(t)
    for _, __ in pairs(t) do
        return false
    end
    return true
end

function table.pack(...)
    return {...}
end

function table.unpack(t)
    return unpack(t, 1, #t)
end

function table.copy(t)
    local copy = {}
    for k, v in pairs(t) do
        copy[k] = v
    end
    return copy
end

function table.deep_copy(t)
    if t == nil then
        return nil
    end
    local copy = {}
    for k, v in pairs(t) do
        if type(v) == "table" then
            copy[k] = table.deep_copy(v)
        else
            copy[k] = v
        end
    end

    setmetatable(copy, table.deep_copy(getmetatable(t)))
    return copy
end

function table.concat_array(...)
    local t = {}
    local i = 0
    for _, item in ipairs({...}) do
        if type(item) == "table" then
            for _, v in ipairs(item) do
                t[i] = v
                i = i + 1
            end
        end
    end
    return t
end

return table