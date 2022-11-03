
function Avatar:on_init()
    print("Avatar:on_init: ", self.id)
    print("level: ", self.level)
    print("items: ", self.items)
end

function Avatar:on_destroy()
    print("Avatar:on_destroy: ", self.id)
end

function Avatar:show_popup_message(msg)
    print("Avatar:show_popup_message: ", msg)
    self.server.use_item(1, 2)
end

function Avatar:on_update_level(old)
    print("on level update old: ", old, ", new: ", self.level)
end