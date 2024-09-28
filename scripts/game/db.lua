local const = require("const")

local db = {}

-- 使用该函数创建filter
function db.set_filter(t, key, value)
    local length = #t
    t[length + 1] = {key, value}
    return t
end

function db.query_account(filter, cb)
    rpg.executeDBRawCommand(const.db_type_project, const.db_query_one, const.db_database_common, const.db_collection_account, filter, {}, function(data, err)
        cb(data, err)
    end)
end

function db.update_account(filter, info, cb)
    rpg.executeDBRawCommand(const.db_type_project, const.db_update_one, const.db_database_common, const.db_collection_account, filter, info, function(data, err)
        cb(data, err)
    end)
end

return db