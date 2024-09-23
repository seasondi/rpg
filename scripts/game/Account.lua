local db = require("db")

function Account:on_created()
    print("Account:on_created")
end

function Account:on_destroy()
    print("Account:on_destroy")
    rpg.callStub("RoleStub", "account_unregister", self.id)
end

function Account:on_get_client()
    print("Account:on_get_client")
end

function Account:login(avtId, client, login_info)
    print("client: ", client)
    local success = rpg.setConnInfo(client, self.id, true)
    if success ~= true then
        self:on_login_failed()
        return
    end
    self.account = login_info.account
    rpg.callStub("RoleStub", "account_register", self.id, self.account)

    if avtId == 0 then
        local id = rpg.createEntityLocally("Avatar")
        if id == 0 then
            self:on_login_failed()
            return
        else
            avt = rpg.entities[id]
            avt:save()
            db.update_account(db.set_filter({}, "account", login_info.account), { ["entityId"] = id}, function(_, err)
                if err ~= nil then
                    self:on_login_failed(id)
                    return
                end
                self:on_login_success(id, client, login_info)
            end)
        end
    else
        avt = rpg.entities[avtId]
        if avt ~= nil then
            self:on_login_success(avt.id, client, login_info)
        else
            rpg.loadEntityFromDB(avtId, function(id, err)
                if err ~= nil then
                    self:on_login_failed(id)
                    return
                end
                self:on_login_success(id, client, login_info)
            end)
        end
    end
end

function Account:on_login_success(avtId, client, login_info)
    print("on_login_success avtId: ", avtId, ", login_info: ", login_info)
    avt = rpg.entities[avtId]
    avt.account_id = self.id
    rpg.setConnInfo(client, self.id, false)
    rpg.setConnInfo(client, avt.id, true)
end

function Account:on_login_failed(avtId)
    print("on_login_failed, avtId: ", avtId)
    self.client.show_popup_message("login failed")
    if avtId ~= nil and avtId ~= 0 then
        rpg.callEntity(avtId, "destroy", false)
    end
    self:destroy()
end