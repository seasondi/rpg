local const = {}
local temp = {}

local mt = {
    __newindex = function(t, k, v)
        if not temp[k] then
            temp[k] = v
        else
            error("cannot change const key " .. k)
        end
    end,
    __index = temp
}

setmetatable(const, mt)

-- ----------------------------------------------------常量定义-----------------------------------------------------------------------
--数据库类型
const.db_type_project = 0 --项目库

--数据库操作类型
const.db_query_one = 0 --查询单条数据
const.db_update_one = 1 --更新单条数据
const.db_replace_one = 2 --替换单条数据
const.db_delete_one = 3 --删除单条数据
const.db_query_many = 4 --查询多条数据
const.db_delete_many = 5 --删除多条数据

const.db_database_common = "common" --common库名称
const.db_collection_account = "account" --account表

const.account_is_logging = "ACCOUNT_IS_LOGGING" --账号正在登录中
const.account_or_password_error = "ACCOUNT_OR_PASSWORD_ERROR" --账号或密码错误
const.login_failed = "LOGIN_FAILED" --登录失败

return const
