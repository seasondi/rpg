local log = require("logger")

function Avatar:use_item(exposed, item_id, item_num)
    log.debug("use_item item_id: ", item_id, ", item_num: ", item_num, ", exposed: ", exposed)
    self.items[item_id] = {["item_id"] = item_id, ["item_num"] = item_num}
    self.items[1001] = nil
    self.items[1000] = 3
    self.client.on_item_used(item_id)
end