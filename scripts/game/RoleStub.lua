local db = require("db")
local common = require("common")
local const = require("const")

function RoleStub:on_created()
    print("RoleStub:on_created ", self.id)
    self.account_map = {}  -- account -> {avatar_id = id}
    self.avatar_map = {} -- {avatar_id = account}
end

function RoleStub:on_destroy()
    print("RoleStub on_destroy")
end

function RoleStub:entry(client, login_info)
    print("call entry with info: ", login_info, " from ", client)
    if common.False(login_info.account) or common.False(login_info.password) then
        client:error(const.account_or_password_error)
        return
    end

    local account = login_info.account
    local password = login_info.password

    if self.account_map[account] ~= nil then --有值说明账号已在登录了
        local avatar_id = self.account_map[account].avatar_id
        local account_id = self.account_map[account].account_id
        if common.True(avatar_id) then -- avatar已创建
            rpg.callEntity(account_id, "login", client, avatar_id, login_info)
        else
            -- avatar尚未创建
            client:error(const.account_is_logging)
        end
        return
    end

    print("load account ", account, " from db")
    self.account_map[account] = {}
    db.query_account(db.set_filter({}, "account", account), function(data, err)
        if err ~= nil then
            print("load account from db error: ", error)
            self:on_login_failed(client, account)
        else
            if table.empty(data) then
                self:create_account(client, login_info)
            else
                if data.password ~= password then
                    print("password not match, account: ", account, ", password: ", password)
                    self:on_login_failed(client, account)
                else
                    self:on_login(client, data.entityId, login_info)
                end
            end
        end
    end)
end

function RoleStub:create_account(client, loginInfo)
    local info = {
        ["account"] = loginInfo.account,
        ["password"] = loginInfo.password,
    }
    db.update_account(db.set_filter({}, "account", info.account), info, function(data, err)
        if err ~= nil then
            print("add db account error: ", error)
            self:on_login_failed(client, info.account)
        else
            self:on_login(client, 0, loginInfo)
        end
    end)
end

function RoleStub:on_login_failed(client, account)
    print("on_login_failed account: ", account)
    client:error(const.login_failed)
end

function RoleStub:on_login(client, avatar_id, login_info)
    rpg.createEntityAnywhere("Account", function(id, error)
        if error ~= nil then
            print("create Account error: ", error, ", login_info: ", login_info, ", avatar_id: ", avatar_id)
            self:on_login_failed(client, login_info.account)
            return
        end
        rpg.callEntity(id, "login", client, avatar_id or 0, login_info)
    end)
end

function RoleStub:avatar_register(avatar_id, account_id, account)
    print("avatar: ", avatar_id, " register, account_id: ", account_id, ", account: ", account)
    self.avatar_map[avatar_id] = account
    self.account_map[account].avatar_id = avatar_id
    self.account_map[account].account_id = account_id
end

function RoleStub:avatar_unregister(avatar_id)
    print("avatar: ", avatar_id, " unregister")
    local account = self.avatar_map[avatar_id]
    self.avatar_map[avatar_id] = nil
    self.account_map[account] = nil
end