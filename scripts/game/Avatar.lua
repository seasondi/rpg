local log = require("logger")

Avatar.isAvatar = true

function Avatar:on_init()
    print("call Avatar:on_init ", self.id)
end

function Avatar:on_get_client()
    print("on_get_client: ", self.id, ", account_id: ", self.account_id)

    rpg.callStub("RoleStub", "avatar_register", self.id, self.account_id)

    if self.is_new_role then
        self.is_new_role = false
    end
    self.level = 10
    self.client.show_popup_message("avatar get client")
    self.client.test({[1]=2, [2]=3,[5]=10}, {1,9, 10, 20}, {id=10, name="1bc", cast=false, arr={9,7,8}, map={[10]=2, [20]=3}})
end

function Avatar:test(_, t1, t2, skill_info)
    print("t1: ", t1, ", t2: ", t2, ", skill: ", skill_info)
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