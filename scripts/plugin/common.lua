local common = {}

function common.True(x)
    if x == nil then
        return false
    elseif type(x) == "number" then
        return x ~= 0
    elseif type(x) == "string" then
        return x ~= ""
    elseif type(x) == "bool" then
        return x
    else
        return true
    end
end

function common.False(x)
    return common.True(x) == false
end

return common