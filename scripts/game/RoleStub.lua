local log = require("logger")
local db = require("db")
local common = require("common")
local const = require("const")
local redis = require("redis")

function RoleStub:on_init()
    print("RoleStub:on_init ", self.id)
    self.account_map = {}  -- account -> {account_id = account_id, avatar_id = avatar_id}
    self.account_id_map = {} -- account_id -> account
    self.avatar_map = {} -- avatar_id -> account_id
end

function RoleStub:on_destroy()
    print("RoleStub on_destroy")
end

function RoleStub:entry(client, login_info)
    print("call entry with info: ", detail(login_info), " from ", client)
    if common.False(login_info.account) or common.False(login_info.password) then
        client:error(const.account_or_password_error)
        return
    end

    local account = login_info.account
    local password = login_info.password

    if common.True(self.account_map[account]) then --有值说明账号已在登录了
        local account_id = self.account_map[account].account_id
        local avt_id = self.account_map[account].avatar_id
        if common.True(account_id) then -- account已创建
            if common.True(avt_id) then -- avatar已创建,直接走登录
                rpg.callEntity(account_id, "login", avt_id, client, login_info)
            else -- avatar尚未创建, 正在登录中
                client:error(const.account_is_logging)
            end
        else --account没创建
            if common.True(avt_id) then --avatar没销毁掉,需排查
                client:error(const.login_failed)
            else -- avatar没创建
                client:error(const.account_is_logging)
            end
        end
        return
    end

    print("load account ", account, " from db")
    self.account_map[account] = {}
    db.query_account(db.set_filter({}, "account", account), function(data, err)
        if err ~= nil then
            print("load account from db error: ", error)
            self:on_login_failed(account)
        else
            if table.empty(data) then
                self:createAccount(client, login_info)
            else
                if data.password ~= password then
                    print("password not match, account: ", account, ", password: ", password)
                    self:on_login_failed(account)
                else
                    self:on_login(client, data.entityId, login_info)
                end
            end
        end
    end)
end

function RoleStub:createAccount(client, loginInfo)
    local info = {
        ["account"] = loginInfo.account,
        ["password"] = loginInfo.password,
    }
    db.update_account(db.set_filter({}, "account", info.account), info, function(data, err)
        if err ~= nil then
            print("add db account error: ", error)
            self:on_login_failed(info.account)
        else
            self:on_login(client, 0, loginInfo)
        end
    end)
end

function RoleStub:on_login_failed(account)
    print("on_login_failed account: ", account)
end

function RoleStub:on_login(client, avtId, loginInfo)
    rpg.createEntityAnywhere("Account", function(id, error)
        if error ~= nil then
            print("create Account error: ", error)
            self:on_login_failed(loginInfo.account)
            return
        end
        rpg.callEntity(id, "login", avtId or 0, client, loginInfo)
    end)
end

function RoleStub:avatar_register(avt_id, account_id)
    log.debug("avatar: ", avt_id, " register, account_id: ", account_id)
    self.avatar_map[avt_id] = account_id
    local account = self.account_id_map[account_id]
    if account ~= nil then
        if self.account_map[account] ~= nil then
            self.account_map[account].avatar_id = avt_id
        else
            self.account_map[account] = {
                ["account_id"] = account_id,
                ["avatar_id"] = avt_id
            }
        end
    end
end

function RoleStub:avatar_unregister(avt_id)
    log.debug("avatar: ", avt_id, " unregister")
    local account_id = self.avatar_map[avt_id]
    self.avatar_map[avt_id] = nil
    if common.True(account_id) then
        local account = self.account_id_map[account_id]
        if common.True(account) then
            self.account_map[account] = nil
        end
        self.account_id_map[account_id] = nil
    end
end

function RoleStub:account_register(entity_id, account)
    log.debug("account: ", entity_id, " register, login account: ", account)
    self.account_id_map[entity_id] = account
    if self.account_map[account] ~= nil then
        self.account_map[account].account_id = entity_id
    else
        self.account_map[account] = {
            ["account_id"] = entity_id
        }
    end
end

function RoleStub:account_unregister(entity_id)
    log.debug("account: ", entity_id, "  unregister")
    local account = self.account_id_map[entity_id]
    if common.True(account) and common.True(self.account_map[account]) then
        self.account_map[account].account_id = nil
    end
end