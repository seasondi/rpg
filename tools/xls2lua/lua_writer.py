import os

import const


def _key_text(key):
    return '["{}"]'.format(key)


def _value_text(value):
    if isinstance(value, list):
        return "{" + ", ".join([_value_text(v) for v in value]) + "}"
    elif isinstance(value, dict):
        text = "{"
        for k, v in value.items():
            text += "[{}] = {}, ".format(_value_text(k), _value_text(v))
        text += "}"
        return text
    elif isinstance(value, str):
        return '"{}"'.format(value)
    elif isinstance(value, bool):
        return "true" if value else "false"
    else:
        return str(value)


# 格式化输出内容
# tabs: 制表符的数量
# expand_line: 是否换行
# value_in_dict: value的内容是否放到{}中
def _format_key_value(key, value, tabs, expand_line, text_model=const.TEXT_MODEL_WITHIN_DICT):
    s = ""
    expand_line_context = expand_line and "\n" or ""
    out_tab_string = "\t" * tabs

    s += out_tab_string + key + " = "
    if text_model != const.TEXT_MODEL_WITHOUT_DICT:
        s += "{"
    s += expand_line_context + value
    if expand_line:
        s += expand_line_context + out_tab_string
    if text_model != const.TEXT_MODEL_WITHOUT_DICT:
        s += "}"
    return s


# 返回key1=value1, key2=value2,...的形式
def _values_to_key_value_text(values: list):
    text = ""
    for value in values:
        text += value.key_value_text() + ", "
    return text


# 返回value1,value2,value3...的形式
def _values_to_value_text(values: list):
    text = ""
    index = 0
    for value in values:
        if index != 0:
            text += ", "
        text += value.value_text()
        index += 1
    return text


class LuaWriterData:
    def __init__(self, name, value):
        self._name = name
        self._value = value

    @property
    def name(self):
        return self._name

    @property
    def value(self):
        return self._value

    def as_key_text(self):
        if isinstance(self._value, str):
            return _key_text(self._value)
        else:
            return "[{}]".format(self._value)

    def name_text(self):
        return _key_text(self._name)

    def value_text(self):
        return _value_text(self._value)

    def key_value_text(self):
        return "{} = {}".format(self.name_text(), self.value_text())


class KvWriter:
    def __init__(self, k, tabs, expand):
        self.key = k
        self.children = []  # 子writer
        self.content = ""  # 最内层的值,数据部分
        self.tabs = tabs  # tab数量
        self.expand = expand  # 是否展开大括号
        self.content_model = const.TEXT_MODEL_WITHIN_DICT  # 数据部分是否放到字典内

    def set_content(self, text, content_model):
        self.content = text
        self.content_model = content_model

    def add_child(self, child):
        self.children.append(child)

    def dump(self):
        if len(self.children) == 0:
            return _format_key_value(self.key, self.content, self.tabs, False, self.content_model)

        values = []
        for child in self.children:
            values.append(child.dump())

        val_str = ""
        for i in range(len(values)):
            val_str += values[i]
            if i + 1 < len(values):
                val_str += ",\n"
        return _format_key_value(self.key, val_str, self.tabs, self.expand)


class LuaWriter:
    def __init__(self, path, lua_file_name):
        if not path:
            raise Exception("未指定lua文件路径")
        if not lua_file_name:
            raise Exception("未指定lua文件名")
        if not os.path.exists(path):
            os.makedirs(path, mode=644, exist_ok=True)

        self._path = path  # --导出路径
        self._lua_file_name = lua_file_name  # --导出文件名
        if not self._lua_file_name.endswith(".lua"):
            self._lua_file_name += ".lua"
        self._file = os.path.join(self._path, self._lua_file_name)
        # 每个主键对应一个层级KvWriter
        self._all_writers = {}  # 所有的KvWriter
        self._level_writers = {}  # 最后一个层级的KvWriter

    @property
    def file_name(self):
        return self._lua_file_name

    def write(self, keys: list[LuaWriterData], values: list[LuaWriterData], write_type=const.WRITE_TYPE_DICT, text_model=const.TEXT_MODEL_WITHIN_DICT):
        child_writer = None
        all_key_str = "|".join([v.as_key_text() for v in keys])

        index, length = 0, len(keys)
        for key in reversed(keys):
            key_text = key.as_key_text()
            if index == 0:
                key_text = all_key_str
            if key_text not in self._all_writers:
                writer = KvWriter(key.as_key_text(), length - index, index != 0)
                if write_type == const.WRITE_TYPE_DICT:
                    writer.set_content(_values_to_key_value_text(values), text_model)
                elif write_type == const.WRITE_TYPE_NO_VALUE_NAME:
                    if text_model == const.TEXT_MODEL_WITHOUT_DICT and len(values) != 1:
                        raise Exception("write file {} failed, values length {} != 1, text model cannot be TEXT_MODEL_WITHOUT_DICT".format(
                            self._lua_file_name, len(values)))
                    writer.set_content(_values_to_value_text(values), text_model)
                self._all_writers[key_text] = writer

            writer = self._all_writers[key_text]
            if child_writer is not None:
                writer.add_child(child_writer)
            child_writer = writer
            if index == length - 1:
                self._level_writers[key_text] = writer
            index += 1

    def dump_to_file(self):
        text = ""
        index, length = 0, len(self._level_writers)
        for writer in self._level_writers.values():
            text += writer.dump()
            if index + 1 < length:
                text += ",\n"
            index += 1
        data = _format_key_value("local data", text, 0, True)
        # print(data)
        data += "\n\nreturn data"
        with open(self._file, "w", encoding="utf-8") as f:
            f.write(data)
