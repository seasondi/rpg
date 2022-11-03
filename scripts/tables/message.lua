local data = {
	[1] = {["id"] = 1, ["enum"] = "ITEM_NOT_ENOUGH", ["context"] = "物品不足", ["isShow"] = true, ["f"] = 1.0, ["t"] = {[1] = "2", [2] = 3, }, ["item_ids"] = {1, 2, 3}, ["item_map"] = {[1] = 2, [2] = 3, [3] = 4, }, },
	[2] = {["id"] = 2, ["enum"] = "ITEM_FULL", ["context"] = "物品已满", ["f"] = 1.24, ["t"] = {1, 2, 3, 4}, ["item_ids"] = {2, 3, 4}, ["item_map"] = {[2] = 3, [3] = 4, [4] = 5, }, },
	[32] = {["id"] = 32, ["enum"] = "ITEM_NOT_ENOUGH2", ["context"] = "等级不足", ["f"] = 0.23, ["t"] = {"a", "b", "c"}, ["item_ids"] = {3, 4, 5}, ["item_map"] = {[3] = 4, [4] = 5, [5] = 6, }, },
	[42] = {["id"] = 42, ["enum"] = "LEVEL_MAX", ["context"] = "等级最大值", ["t"] = {{1, 2}, {"a", "b"}}, ["item_ids"] = {4, 5, 6}, ["item_map"] = {[4] = 5, [5] = 6, [6] = 7, }, },
	[20] = {["id"] = 20, ["enum"] = "ITEM_NOT_ENOUGH3", ["context"] = "物品不足", ["isShow"] = true, ["f"] = 1.24, ["t"] = {[1] = "2", [2] = 3, }, ["item_ids"] = {1, 2, 3}, ["item_map"] = {[1] = 2, [2] = 3, [3] = 4, }, }
}

return data