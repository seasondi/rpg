import os
import const


# 是否主键
def is_primary(sheet, col):
    return sheet.cell_value(const.EXCEL_HEADER_ROW_INDEX_PRIMARY, col) == "Y"


# 根据字段名判断是否是主键
def is_primary_name(sheet, name):
    for col in range(const.EXCEL_COL_START, sheet.ncols):
        if get_key(sheet, col) == name:
            return is_primary(sheet, col)
    return False


# 获取字段英文名
def get_key(sheet, col):
    return sheet.cell_value(const.EXCEL_HEADER_ROW_INDEX_KEY, col)


# 获取字段类型
def get_type(sheet, col):
    return sheet.cell_value(const.EXCEL_HEADER_ROW_INDEX_TYPE, col)


# 获取字段是否必填
def is_require(sheet, col):
    return sheet.cell_value(const.EXCEL_HEADER_ROW_INDEX_REQUIRE, col) == "Y"


# 获取字段是否需要导出到指定目标
def is_export(sheet, col, target=None):
    export = sheet.cell_value(const.EXCEL_HEADER_ROW_INDEX_TARGET, col)
    if target:
        return export == target
    else:
        return True if export in const.ALL_EXPORT_TARGETS else False


# 根据字段名判断是否需要导出
def is_export_name(sheet, name, target):
    for col in range(const.EXCEL_COL_START, sheet.ncols):
        if get_key(sheet, col) == name:
            return is_export(sheet, col, target)
    return False


# 判断字段是否表格原始字段(预处理阶段可能会加入新字段)
def is_sheet_origin_field(sheet, name):
    for col in range(const.EXCEL_HEADER_ROW_INDEX_KEY, sheet.ncols):
        if get_key(sheet, col) == name:
            return True
    return False


# 获取单元格名称
def get_cell_name(row, col):
    col_chr = chr(col + const.COL_A_ASCII)
    return "({}, {})".format(row + 1, col_chr)


# 清空导表临时目录
def cleanup_temp_dir():
    if os.path.exists(const.TEMP_CLIENT_DIR):
        ls = os.listdir(const.TEMP_CLIENT_DIR)
        for i in ls:
            file = os.path.join(const.TEMP_CLIENT_DIR, i)
            if os.path.isfile(file):
                os.remove(file)
            else:
                os.removedirs(file)

    if os.path.exists(const.TEMP_SERVER_DIR):
        ls = os.listdir(const.TEMP_SERVER_DIR)
        for i in ls:
            file = os.path.join(const.TEMP_SERVER_DIR, i)
            if os.path.isfile(file):
                os.remove(file)
            else:
                os.removedirs(file)


def default_value(value_type):
    if value_type == const.VALUE_TYPE_FLOAT:
        return 0.0
    elif value_type == const.VALUE_TYPE_BOOL:
        return False
    elif value_type == const.VALUE_TYPE_INT:
        return 0
    elif value_type == const.VALUE_TYPE_STR:
        return ""
    elif value_type == const.VALUE_TYPE_TABLE:
        return {}
    else:
        raise Exception("gen default value unknown type {}".format(value_type))
