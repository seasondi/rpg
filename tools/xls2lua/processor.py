import const
import common
import checker
import excel_value_parser
import lua_writer
import user_behavior


# 预处理器
class BeforeProcessor:
    def __init__(self, file_name, sheet, all_sheets):
        self.sheet_file = file_name
        self.sheet = sheet
        self.all_sheets = all_sheets
        self.all_keys = {}

    def process(self, target):
        checker.check_sheet_headers(self.sheet_file, self.sheet)
        result = []
        for row in range(const.EXCEL_HEADER_ROWS, self.sheet.nrows):
            result.append(self._process_row(row))
        funcs = user_behavior.client_before_process_behaviors if target == const.TARGET_CLIENT \
            else user_behavior.server_before_process_behaviors
        for f in funcs:
            if self.sheet.name in f.ref_sheets:
                try:
                    f({self.sheet.name: result})
                except Exception as ex:
                    raise Exception("导表预处理失败, 文件: {}, 表格: {}, 预处理函数: {}, 错误信息: {}".format(
                        self.sheet_file, self.sheet.name, f.__name__, ex
                    ))
        return result

    def _process_row(self, row):
        checker.check_row(self.sheet_file, self.sheet, row, self.all_sheets, self.all_keys)
        result = {}
        for col in range(const.EXCEL_COL_START, self.sheet.ncols):
            if not common.is_export(self.sheet, col):
                continue
            value = self.sheet.cell_value(row, col)
            value_type = common.get_type(self.sheet, col)
            if value == "":
                value = common.default_value(value_type)
            result[common.get_key(self.sheet, col)] = excel_value_parser.convert_excel_value_to_python(value, value_type)
        return result


# 后处理器
class AfterProcessor:
    def __init__(self, sheets, data, target):
        self.sheets = sheets  # 所有表格原始数据
        self.data = data  # 所有表格预处理后数据
        self.target = target

    def process(self):
        funcs = user_behavior.client_after_process_behaviors if self.target == const.TARGET_CLIENT \
            else user_behavior.server_after_process_behaviors
        for f in funcs:
            data = {}
            for name in f.ref_sheets:
                if name in self.data:
                    data[name] = self.data[name]
                else:
                    raise Exception("导表后处理失败, 后处理函数: {}, 关注的表格: {} 不存在".format(f.__name__, name))
            try:
                f(data)
            except Exception as ex:
                raise Exception("导表后处理失败, 后处理函数: {}, 错误信息: {}".format(f.__name__, ex))


# 导表处理器
class ExportProcessor:
    def __init__(self, writer, sheet, export_data, target):
        self.writer = writer  # 文件写对象
        self.sheet = sheet  # 原始表格
        self.data = export_data  # 预处理后的数据
        self.target = target  # 导出目标

    def process(self):
        for row_data in self.data:
            keys, values = [], []
            for name, value in row_data.items():
                # 表格中未填写导出目标的字段不导出(预处理阶段加入的新字段仍然需要导出)
                if not common.is_export_name(self.sheet, name, self.target) and common.is_sheet_origin_field(self.sheet, name):
                    continue
                data = lua_writer.LuaWriterData(name, value)
                if common.is_primary_name(self.sheet, name):
                    keys.append(data)
                values.append(data)
            self.writer.write(keys, values)
        self.writer.dump_to_file()

