function Account:on_created()
    print("Account:on_created: ", self.id)
end

function Account:on_destroy()
    print("Account:on_destroy: ", self.id)
end

function Account:show_popup_message(msg)
    print("Account:show_popup_message: ", msg)
end