function Account:on_init()
    print("Account:on_init: ", self.id)
end

function Account:on_destroy()
    print("Account:on_destroy: ", self.id)
end

function Account:show_popup_message(msg)
    print("Account:show_popup_message: ", msg)
end