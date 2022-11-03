import excel_parser

if __name__ == '__main__':
    parser = excel_parser.ExcelParser("./excels")
    try:
        parser.export_to_lua()
        print("导出成功!!!!")
    except Exception as ex:
        print(ex)
