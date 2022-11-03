local log = require("logger")

function Avatar:on_item_used(item_id)
    log.debug("on_item_used item_id: ", item_id)
end

function Avatar:on_update_items(old)
    print("on_items, old: ", old, ", new: ", self.items)
end