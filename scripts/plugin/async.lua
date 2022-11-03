--该功能目前使用限制较多,暂不可使用

local async_funcs = {}

function make_async(func_name)
    local func = async_funcs[func_name]
    if type(func) ~= "function" then
        error("make_async: function[" .. func_name .. "] not defined")
    end
    local meta = {
        __call = function(mt, ...)
            local co = coroutine.create(function(...)
                mt.func(...)
            end)
            local ok, errMsg = coroutine.resume(co, ...)
            if ok ~= true then
                error(errMsg)
            end
        end
    }
    local ct = {
        ["name"] = func_name,
        ["func"] = func
    }
    setmetatable(ct, meta)
    async_funcs[func_name] = ct
end

function await(func, ...)
    local parent = coroutine.running()
    if type(parent) ~= "thread" then
        error("can not call await outside of coroutine")
    end

    --启用新协程执行函数
    local co = coroutine.create(function(...)
        local current = coroutine.running()
        local _, errMsg = pcall(func, current, table.unpack({...}))
        local r = {coroutine.yield(errMsg)}
        if coroutine.status(parent) == "suspended" then
            coroutine.resume(parent, table.unpack(r))
        end
    end)
    local _, errMsg = coroutine.resume(co, ...)
    if errMsg ~= nil then
        coroutine.resume(co)
        error(errMsg)
    else
        --挂起调用者协程,等待返回值
        return table.unpack({coroutine.yield()})
    end
end