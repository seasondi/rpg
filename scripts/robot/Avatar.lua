
function Avatar:on_init()
    print("Avatar:on_init: ", self.id)
end

function Avatar:on_destroy()
    print("Avatar:on_destroy: ", self.id)
end

function Avatar:show_popup_message(msg)
    print("Avatar:show_popup_message: ", msg)
end

function Avatar:test(t1, t2, skill)
    print("Avatar:test")
    self.server.test(t1, t2, skill)
end