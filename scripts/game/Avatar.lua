local log = require("logger")

Avatar.isAvatar = true

function Avatar:on_init()
    print("call Avatar:on_init ", self.id)
    self.items[1001] = {["item_id"] = 1001, ["item_num"] = 2}
end

function Avatar:on_get_client()
    print("on_get_client: ", self.id, ", account_id: ", self.account_id)

    rpg.callStub("RoleStub", "avatar_register", self.id, self.account_id)

    if self.is_new_role then
        self.is_new_role = false
    end
    self.level = 10
    self.client.show_popup_message("avatar get client")
end

function Avatar:on_lose_client()
    print("on_lose_client: ", self.id)
end

function Avatar:on_destroy()
    log.info("call Avatar:on_destroy, id: ", self.id)
end

-- 存盘的entity存盘成功后回调
function Avatar:on_final()
    log.info("call Avatar:on_final, id: ", self.id)
    rpg.callStub("RoleStub", "avatar_unregister", self.id)
end