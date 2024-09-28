local db = require("db")
local const = require("const")

function Account:on_created()
    print("Account:on_created")
end

function Account:on_destroy()
    print("Account:on_destroy")
end

function Account:on_get_client()
    print("Account:on_get_client")
end

function Account:login(client, avatar_id, login_info)
    print("account login, client: ", client, ", avatar_id: ", avatar_id, ", login_info: ", login_info)
    self.account = login_info.account

    if avatar_id == 0 then
        local id = rpg.createEntityLocally("Avatar")
        if id == 0 then
            self:on_login_failed(client, id)
            return
        else
            -- save new avatar
            local avt = rpg.entities[id]
            avt:save()

            -- associate avatar_id to account
            db.update_account(db.set_filter({}, "account", login_info.account), { ["entityId"] = id}, function(_, err)
                if err ~= nil then
                    self:on_login_failed(client, id, err)
                    return
                end
                self:on_login_success(client, id, login_info)
            end)
        end
    else
        local avt = rpg.entities[avatar_id]
        if avt ~= nil then
            self:on_login_success(client, avt.id, login_info)
        else
            rpg.loadEntityFromDB(avatar_id, function(id, err)
                if err ~= nil then
                    self:on_login_failed(client, id, err)
                    return
                end
                self:on_login_success(client, id, login_info)
            end)
        end
    end
end

function Account:on_login_success(client, avatar_id, login_info)
    print("on_login_success avatar_id: ", avatar_id, ", login_info: ", login_info)
    local avt = rpg.entities[avatar_id]
    avt.account = login_info.account
    avt.account_id = self.id
    rpg.callStub("RoleStub", "avatar_register", avt.id, avt.account_id, avt.account)
    rpg.setConnInfo(client, avatar_id)
end

function Account:on_login_failed(client, avatar_id, error)
    print("on_login_failed, avatar_id: ", avatar_id, ", error: ", error)
    client:error(const.login_failed)
    local avt = rpg.entities[avatar_id]
    if avt ~= nil then
        avt:destroy(false)
    end
    self:destroy(false)
end