import const
import common
import re
import excel_value_parser


def _ref(sheet_name, field_name):
    if not isinstance(sheet_name, str):
        raise Exception("_ref的第一个参数需要填字符串")
    if not isinstance(field_name, str):
        raise Exception("_ref的第二个参数需要填字符串")
    return const.DEPEND_NAME_REF, sheet_name, field_name


def _key_ref(sheet_name, field_name):
    if not isinstance(sheet_name, str):
        raise Exception("_key_ref的第一个参数需要填字符串")
    if not isinstance(field_name, str):
        raise Exception("_key_ref的第二个参数需要填字符串")
    return const.DEPEND_NAME_KEY_REF, sheet_name, field_name


# 检查value是否在表格ref_sheet的ref_key字段中有值
def check_ref(ref_sheet, ref_key, value):
    ref_col = None
    for col in range(const.EXCEL_COL_START, ref_sheet.ncols):
        if ref_sheet.cell_value(const.EXCEL_HEADER_ROW_INDEX_KEY, col) == ref_key:
            ref_col = col
            break

    if not ref_col:
        return False

    find = False
    for row in range(const.EXCEL_HEADER_ROWS, ref_sheet.nrows):
        if ref_sheet.cell_value(row, ref_col) == value:
            find = True
            break

    return find


# 检查字典的key或者数组的元素的关联数据
def check_key_ref(ref_sheet, ref_key, value):
    table = excel_value_parser.convert_excel_value_to_python(value, const.VALUE_TYPE_TABLE)
    if isinstance(table, dict):
        for key in table.keys():
            if not check_ref(ref_sheet, ref_key, key):
                return False
        return True
    elif isinstance(table, list):
        for item in table:
            if not check_ref(ref_sheet, ref_key, item):
                return False
        return True
    else:
        raise Exception("_key_ref只能作用于字典或者数组")


# 检查表格头(只检查需要导出的列)
def check_sheet_headers(sheet_file, sheet):
    if sheet.nrows < const.EXCEL_HEADER_ROWS:
        raise Exception("文件:{} 表格:{} 行数必须不小于 {}".format(
            sheet_file, sheet.name, const.EXCEL_HEADER_ROWS))

    keys = []
    has_primary_keys = False
    for i in range(const.EXCEL_COL_START, sheet.ncols):
        # 检查导出目标
        val = sheet.cell_value(const.EXCEL_HEADER_ROW_INDEX_TARGET, i)
        if val != "" and val not in const.ALL_EXPORT_TARGETS:
            raise Exception("文件:{} 表格:{} 单元格: {} 未知的值:'{}', 支持的值有{}".format(
                sheet_file, sheet.name,
                common.get_cell_name(const.EXCEL_HEADER_ROW_INDEX_TARGET, i), val,
                const.ALL_EXPORT_TARGETS
            ))

        need_export = val in const.ALL_EXPORT_TARGETS
        if not need_export:
            continue

        # 字段中文名
        name = sheet.cell_value(const.EXCEL_HEADER_ROW_INDEX_NAME, i)
        if not name:
            raise Exception("文件:{} 表格:{} 单元格: {} 字段未填写中文名".format(
                sheet_file, sheet.name,
                common.get_cell_name(const.EXCEL_HEADER_ROW_INDEX_NAME, i)
            ))

        # 检查字段英文名
        val = sheet.cell_value(const.EXCEL_HEADER_ROW_INDEX_KEY, i)
        if re.match(const.PATTERN_LETTER_NAME, val, re.I) is None:
            raise Exception("文件:{} 表格:{} 单元格: {} 字段英文名:'{}'格式错误, 字段名: {}".format(
                sheet_file, sheet.name,
                common.get_cell_name(const.EXCEL_HEADER_ROW_INDEX_KEY, i), val, name
            ))
        if val in keys:
            raise Exception("文件:{} 表格:{} 单元格: {} 字段英文名:'{}'重复, 字段名: {}".format(
                sheet_file, sheet.name,
                common.get_cell_name(const.EXCEL_HEADER_ROW_INDEX_KEY, i), val, name
            ))
        else:
            keys.append(val)

        # 检查字段类型
        val = sheet.cell_value(const.EXCEL_HEADER_ROW_INDEX_TYPE, i)
        if val not in const.ALL_VALUE_TYPES:
            passed = False
            if val.startswith(const.VALUE_TYPE_FLOAT):
                arr = val.split(',')
                try:
                    decimal = int(arr[1])
                    if const.FLOAT_DECIMAL_MAX >= decimal >= const.FLOAT_DECIMAL_MIN:
                        passed = True
                    else:
                        print("浮点数小数位范围错误, 支持范围: [{}, {}]".format(const.FLOAT_DECIMAL_MIN, const.FLOAT_DECIMAL_MAX))
                except Exception as ex:
                    print("浮点数小数位解析失败: ", ex)
            if not passed:
                raise Exception("文件:{} 表格:{} 单元格: {} 字段类型:'{}'错误, 字段名: {}, 支持的类型有{}".format(
                    sheet_file, sheet.name,
                    common.get_cell_name(const.EXCEL_HEADER_ROW_INDEX_TYPE, i), val,
                    name, str(const.ALL_VALUE_TYPES)
                ))

        # 检查主键
        val = sheet.cell_value(const.EXCEL_HEADER_ROW_INDEX_PRIMARY, i)
        if val != "" and val != "Y":
            raise Exception("文件:{} 表格:{} 单元格: {} 未知的值'{}', 字段名: {}, 仅支持Y或者为空".format(
                sheet_file, sheet.name,
                common.get_cell_name(const.EXCEL_HEADER_ROW_INDEX_PRIMARY, i), val, name
            ))
        if val == "Y":
            has_primary_keys = True

        # 检查是否必填
        val = sheet.cell_value(const.EXCEL_HEADER_ROW_INDEX_REQUIRE, i)
        if val != "" and val != "Y":
            raise Exception("文件:{} 表格:{} 单元格: {} 未知的值'{}', 字段名: {}, 仅支持Y或者为空".format(
                sheet_file, sheet.name,
                common.get_cell_name(const.EXCEL_HEADER_ROW_INDEX_REQUIRE, i), val, name
            ))

    if not has_primary_keys:
        raise Exception("文件:{} 表格:{}, 未指定主键".format(sheet_file, sheet.name))


# 检查指定表格指定行的数据
def check_row(sheet_file, sheet, row, all_sheets, all_keys):
    key_str = ""
    for col in range(const.EXCEL_COL_START, sheet.ncols):
        val = sheet.cell_value(row, col)
        if common.is_primary(sheet, col):
            ty = common.get_type(sheet, col)
            if ty not in const.KEY_VALUE_TYPES and not ty.startswith(const.VALUE_TYPE_FLOAT):
                raise Exception("文件:{} 表格:{} 字段名: {} 不是主键类型, 支持的主键类型包括: {}".format(
                    sheet_file, sheet.name, common.get_key(sheet, col), const.KEY_VALUE_TYPES
                ))
            if key_str != "":
                key_str += "|"
            key_str += str(val)
        # 非主键非必填字段允许为空
        if val == "":
            if not common.is_primary(sheet, col) and not common.is_require(sheet, col):
                continue
            else:
                raise Exception("文件:{} 表格:{} 单元格: {} 不能为空(该值为主键或者必填项)".format(
                    sheet_file, sheet.name, common.get_cell_name(row, col)
                ))
        # 校验类型
        try:
            if not excel_value_parser.check_excel_value(val, common.get_type(sheet, col)):
                raise Exception("文件:{} 表格:{} 单元格:{} 数据'{}' 格式错误".format(
                    sheet_file, sheet.name, common.get_cell_name(row, col), val
                ))
        except Exception as ex:
            raise Exception("文件:{} 表格:{} 单元格:{} 数据'{}' 格式错误: {}".format(
                sheet_file, sheet.name, common.get_cell_name(row, col), val, ex
            ))

        # 校验关联
        depends = sheet.cell_value(const.EXCEL_HEADER_ROW_INDEX_DEPENDS, col)
        for depend in depends.split(";"):
            if not depend:
                continue
            c = compile(depend, "", "eval")
            try:
                name, ref_sheet_name, ref_field_name = eval(c)
                if name == const.DEPEND_NAME_REF:
                    if ref_sheet_name not in all_sheets:
                        raise Exception("文件:{} 表格:{} 单元格:{} _ref依赖的表{}不存在".format(
                            sheet_file, sheet.name, common.get_cell_name(row, col), ref_sheet_name
                        ))
                    if not check_ref(all_sheets[ref_sheet_name], ref_field_name, val):
                        raise Exception("文件:{} 表格:{} 单元格:{} _ref依赖数据检查未通过,请检查该单元格数据是否存在于{}的字段{}中".format(
                            sheet_file, sheet.name, common.get_cell_name(row, col),
                            ref_sheet_name, ref_field_name
                        ))
                elif name == const.DEPEND_NAME_KEY_REF:
                    if ref_sheet_name not in all_sheets:
                        raise Exception("文件:{} 表格:{} 单元格:{} _key_ref依赖的表{}不存在".format(
                            sheet_file, sheet.name, common.get_cell_name(row, col), ref_sheet_name
                        ))
                    if common.get_type(sheet, col) != const.VALUE_TYPE_TABLE:
                        raise Exception("文件:{} 表格:{} 单元格:{}, _key_ref只能作用在table上".format(
                            sheet_file, sheet.name, common.get_cell_name(row, col)
                        ))
                    if not check_key_ref(all_sheets[ref_sheet_name], ref_field_name, val):
                        raise Exception("文件:{} 表格:{} 单元格:{} _key_ref依赖数据检查未通过,请检查该单元格数据的key是否存在于{}的字段{}中".format(
                            sheet_file, sheet.name, common.get_cell_name(row, col),
                            ref_sheet_name, ref_field_name
                        ))
            except Exception as ex:
                raise Exception("文件:{} 表格:{} 单元格:{} 错误: {}".format(
                    sheet_file, sheet.name, common.get_cell_name(const.EXCEL_HEADER_ROW_INDEX_DEPENDS, col), ex
                ))

    # 校验主键
    if key_str in all_keys:
        raise Exception("文件:{} 表格:{} 第{}行主键{}与第{}行重复".format(
            sheet_file, sheet.name, row + 1,
            tuple(key_str.split("|")), all_keys[key_str] + 1
        ))
    else:
        all_keys[key_str] = row
