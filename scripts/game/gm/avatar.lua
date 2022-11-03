require("gm.common")

GMAvatar = {}

-- ---------------------------------------------------------------------------------- --
GMAvatar.add_item = NewGMCommand("添加物品", GMAuthorityDebug,
        GMEntityId(),
        GMNumber("item_id", "物品ID"),
        GMNumber("item_cnt", "物品数量")
)
GMAvatar.add_item.callback = function(ent, item_id, item_cnt)
    print("ent: ", ent, ", item_id: ", item_id, ", item_cnt: ", item_cnt)

    return "物品添加成功"
end

-- ---------------------------------------------------------------------------------- --
GMAvatar.del_item = NewGMCommand("删除物品", GMAuthorityDebug,
        GMEntityId(),
        GMNumber("item_id", "物品ID"),
        GMNumber("item_cnt", "物品数量"),
        GMBool("is_test", "测试")
)
GMAvatar.del_item.callback = function(ent, item_id, item_cnt)
    print("ent: ", ent, ", item_id: ", item_id, ", item_cnt: ", item_cnt)

    return "物品删除成功"
end

-- ---------------------------------------------------------------------------------- --
GMAvatar.kick_user = NewGMCommand("踢玩家下线", GMAuthorityDebug,
        GMEntityId()
)
GMAvatar.kick_user.callback = function(ent)
    ent:destroy()

    return "操作完成"
end