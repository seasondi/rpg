# 导表用户自定义行为
import const
import lua_writer

client_before_process_behaviors = []
client_after_process_behaviors = []

server_before_process_behaviors = []
server_after_process_behaviors = []


def sheet_process(behavior, *sheet_names):
    def decorator_func(func):
        func.ref_sheets = sheet_names
        if behavior & const.CLIENT_BEFORE_PROCESS or behavior & const.SERVER_BEFORE_PROCESS:
            if len(sheet_names) != 1:
                raise Exception("user behavior func {} can only ref 1 sheet, but got {}".format(
                    func.__name__, len(sheet_names)))
            if behavior & const.CLIENT_BEFORE_PROCESS:
                client_before_process_behaviors.append(func)
            elif behavior & const.SERVER_BEFORE_PROCESS:
                server_before_process_behaviors.append(func)
        elif behavior & const.CLIENT_AFTER_PROCESS or behavior & const.SERVER_AFTER_PROCESS:
            if len(sheet_names) < 1:
                raise Exception("user behavior func {} has no ref sheet".format(func.__name__))
            if behavior & const.CLIENT_AFTER_PROCESS:
                client_after_process_behaviors.append(func)
            elif behavior & const.SERVER_AFTER_PROCESS:
                server_after_process_behaviors.append(func)
        else:
            raise Exception("unknown process behavior: {} for sheets: {}".format(behavior, sheet_names))
    return decorator_func


# 将所有字段名前缀为prefix的字段合并为数组
def _array(data: dict, prefix: str):
    r = []
    for k, v in data.items():
        if k.startswith(prefix):
            r.append(v)
    return r


# 将所有前缀为key_prefix的字段与对应的后缀为value_prefix的字段组合成一个字典
def _map(data: dict, key_prefix: str, value_prefix: str):
    r = {}
    for k, v in data.items():
        if k.startswith(key_prefix):
            value_name = value_prefix + k.strip(key_prefix)
            if value_name in data:
                r[v] = data[value_name]
            else:
                raise Exception('_map: 未找到键"{}"对应的值"{}",请检查表格是否存在该字段或者导出目标是否配置'.format(k, value_name))
    return r


# 移除所有包含指定前缀的字段
def _remove_prefix(data: dict, *prefix):
    keys = []
    for k in data.keys():
        for p in prefix:
            if k.startswith(p):
                keys.append(k)
    for k in keys:
        data.pop(k)


def _gen_key_values(data: dict, key_names: list, value_names: list):
    keys, values = [], []
    for key in key_names:
        if key in data:
            keys.append(lua_writer.LuaWriterData(key, data[key]))
    for value in value_names:
        if value in data:
            values.append(lua_writer.LuaWriterData(value, data[value]))

    return keys, values


@sheet_process(const.SERVER_BEFORE_PROCESS, "消息提示表")
def process_a(data_map: dict):
    for row_data in data_map["消息提示表"]:
        row_data["item_ids"] = _array(row_data, "item_id_")
        row_data["item_map"] = _map(row_data, "item_id_", "item_num_")
        _remove_prefix(row_data, "item_id_", "item_num_")


@sheet_process(const.SERVER_AFTER_PROCESS, "消息提示表")
def process_b(data_map: dict):
    print("生成消息提示反转表")
    writer = lua_writer.LuaWriter(const.TEMP_SERVER_DIR, "message_reversed.lua")
    for row_data in data_map["消息提示表"]:
        keys, values = _gen_key_values(row_data, ["enum"], ["id"])
        writer.write(keys, values, write_type=const.WRITE_TYPE_NO_VALUE_NAME, text_model=const.TEXT_MODEL_WITHOUT_DICT)
    writer.dump_to_file()
