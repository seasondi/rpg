import os
import xlrd

import common
import lua_writer
import excel_define
import const
import processor


class ExcelParser:
    def __init__(self, path):
        self.excel_path = path
        self._all_sheets = {}  # 所有表格的sheet_name -> sheet, 原始表格数据
        self._processed_client_data = {}  # 所有客户端表格的sheet_name -> values, 预处理之后的数据
        self._processed_server_data = {}  # 所有服务端表格的sheet_name -> values, 预处理之后的数据
        self._sheet_to_file = {}  # sheet名称到excel名称的映射

        self.load_excels()

    # 加载目录下所有excel文件
    def load_excels(self):
        for root, dirs, files in os.walk(self.excel_path):
            for file in files:
                self._load_excel_file(os.path.join(root, file))

    # 加载指定excel文件
    def _load_excel_file(self, filename):
        if not filename.endswith(".xls"):
            return
        book = xlrd.open_workbook(filename, encoding_override="utf-8")
        for name in book.sheet_names():
            sheet = book.sheet_by_name(name)
            self._all_sheets[name] = sheet
            if name not in self._sheet_to_file:
                self._sheet_to_file[name] = filename
            else:
                raise Exception("加载文件:{} 发现重复的表格名:{}, 已存在于文件:{}".format(filename, name, self._sheet_to_file[name]))

    # 导表预处理
    def _before_process(self, sheet, target):
        # print("\n=======================开始预处理表格[{}]数据============================".format(sheet.name))
        bp = processor.BeforeProcessor(self._sheet_to_file[sheet.name], sheet, self._all_sheets)
        if target == const.TARGET_CLIENT:
            self._processed_client_data[sheet.name] = bp.process(target)
        elif target == const.TARGET_SERVER:
            self._processed_server_data[sheet.name] = bp.process(target)
        else:
            raise Exception("预处理表格[{}], 导出目标[{}]错误".format(sheet.name, target))
        # print("=======================预处理表格[{}]数据完成============================".format(sheet.name))

    # 所有excel导出到lua
    def export_to_lua(self):
        common.cleanup_temp_dir()
        print("\n=======================开始导出表格数据============================")
        for sheet_name, targets in excel_define.export_targets.items():
            if sheet_name not in self._all_sheets:
                print("WARN: 表格\"{}\"不存在, 请检查导出配置文件, 已跳过该配置.".format(sheet_name))
                continue
            for target, file in targets.items():
                self._before_process(self._all_sheets[sheet_name], target)
                self._export_sheet_to_lua(self._all_sheets[sheet_name], target, file)
        print("=======================导出表格数据完成============================")
        self._after_process()

    # 导表后处理
    def _after_process(self):
        print("\n========================开始客户端表格导表后处理==============================")
        pc = processor.AfterProcessor(self._all_sheets, self._processed_client_data, const.TARGET_CLIENT)
        pc.process()
        print("========================客户端表格导表后处理完成==============================")

        print("\n========================开始服务端表格导表后处理==============================")
        ps = processor.AfterProcessor(self._all_sheets, self._processed_server_data, const.TARGET_SERVER)
        ps.process()
        print("========================服务端表格导表后处理完成==============================")

    # 导出指定页签
    def _export_sheet_to_lua(self, sheet, target, file):
        if target == const.TARGET_CLIENT:
            print("正在导出到客户端, 文件: %s 表格: %s, 共%d行%d列" % (
                self._sheet_to_file[sheet.name], sheet.name, sheet.nrows, sheet.ncols))
            writer = lua_writer.LuaWriter(const.TEMP_CLIENT_DIR, file)
            self._export_sheet(sheet, writer, const.TARGET_CLIENT)
            print("文件: %s 表格: %s 导出客户端成功, 文件名: %s" % (self._sheet_to_file[sheet.name], sheet.name, writer.file_name))
        elif target == const.TARGET_SERVER:
            print("正在导出到服务端, 文件: %s 表格: %s, 共%d行%d列" % (
                self._sheet_to_file[sheet.name], sheet.name, sheet.nrows, sheet.ncols))
            writer = lua_writer.LuaWriter(const.TEMP_SERVER_DIR, file)
            self._export_sheet(sheet, writer, const.TARGET_SERVER)
            print("文件: %s 表格: %s 导出服务端成功, 文件名: %s" % (self._sheet_to_file[sheet.name], sheet.name, writer.file_name))

    # 导表
    def _export_sheet(self, sheet, writer, target):
        if target not in [const.TARGET_CLIENT, const.TARGET_SERVER]:
            raise Exception("文件: {} 表格:{} 导出目标{}错误".format(
                self._sheet_to_file[sheet.name], sheet.name, target
            ))
        data = self._processed_client_data if target == const.TARGET_CLIENT else self._processed_server_data
        p = processor.ExportProcessor(writer, sheet, data[sheet.name], target)
        p.process()

    def get_sheet_file(self, sheet_name):
        if sheet_name in self._sheet_to_file:
            return "【{}】所在的文件:【{}】".format(sheet_name, self._sheet_to_file[sheet_name])
        else:
            similar = {}
            for name in self._sheet_to_file.keys():
                match_count = 0
                for ch in sheet_name:
                    if ch != "表" and ch in name:
                        match_count += 1
                if match_count > 0:
                    similar[name] = match_count

            if len(similar) > 0:
                r = sorted(similar.items(), key=lambda x: x[1], reverse=True)
                return "表格不存在, 或许想查询: " + ", ".join(v[0] for v in r[:5])
            else:
                return "表格不存在"
