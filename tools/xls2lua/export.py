import os
import sys
import shutil
import const
import excel_parser


def copy_table_files(path, target):
    if not os.path.exists(target):
        os.makedirs(target)
    for _, _, files in os.walk(path):
        for file in files:
            if not file.endswith(".lua"):
                continue
            shutil.copyfile(path + "/" + file, target + "/" + file)


def parse_args(argv: list):
    d = {}
    for v in argv:
        r = v.split("=")
        if len(r) == 2:
            d[r[0]] = r[1]
    return d


arg_excel_dir = "--excel_path"  # excel路径
arg_find_sheet = "--find_sheet"  # 要查询的sheet名称
arg_client_output = "--client_output"  # 客户端表格输出路径
arg_server_output = "--server_output"  # 服务端表格输出路径

if __name__ == '__main__':
    args = parse_args(sys.argv)
    if arg_excel_dir not in args:
        raise Exception("缺少excel路径, 添加{}=/path/to/excel".format(arg_excel_dir))

    parser = excel_parser.ExcelParser(args[arg_excel_dir])
    if arg_find_sheet in args:
        print(parser.get_sheet_file(args[arg_find_sheet]))
    else:
        if arg_client_output not in args:
            raise Exception("缺少客户端表格输出路径, 添加{}=/path/to/client/table/output".format(arg_client_output))
        if arg_server_output not in args:
            raise Exception("缺少服务端表格输出路径, 添加{}=/path/to/server/table/output".format(arg_server_output))
        print("开始导表, excel路径: ", args[arg_excel_dir],
              ", 客户端表格输出路径: ", args[arg_client_output], ", 服务器表格输出路径: ", args[arg_server_output])
        try:
            parser.export_to_lua()
            copy_table_files(const.TEMP_CLIENT_DIR, args[arg_client_output])
            copy_table_files(const.TEMP_SERVER_DIR, args[arg_server_output])
            print("SUCCESS!!!!")
        except Exception as ex:
            print(ex)
            print("FAILED!!!!")
