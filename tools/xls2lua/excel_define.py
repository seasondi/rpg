import const

# 指定需要导出的sheet以及对应的导出文件名
# 格式: sheet名称: {导出目标: 导出文件名}
export_targets = {
    "消息提示表": {const.TARGET_CLIENT: "message", const.TARGET_SERVER: "message"},
    "物品表": {const.TARGET_CLIENT: "item", const.TARGET_SERVER: "item"},
}
