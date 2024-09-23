local log = require("logger")

function Avatar:on_created()
    print("Avatar:on_created: ", self.id)
end

function Avatar:on_destroy()
    print("Avatar:on_destroy: ", self.id)
end

function Avatar:show_popup_message(msg)
    print("Avatar:show_popup_message: ", msg)
end

function Avatar:test(t1, t2, skill)
    log.info(self, "call Avatar:test")
    self.server.test(t1, t2, skill)
end