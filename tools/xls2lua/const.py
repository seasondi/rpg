COL_A_ASCII = 65  # 字符A ASCII码

EXCEL_HEADER_ROWS = 8  # header行数
EXCEL_HEADER_ROW_INDEX_NAME = 1  # 字段中文名
EXCEL_HEADER_ROW_INDEX_KEY = 2  # 字段英文名
EXCEL_HEADER_ROW_INDEX_TYPE = 3  # 字段类型
EXCEL_HEADER_ROW_INDEX_PRIMARY = 4  # 是否主键
EXCEL_HEADER_ROW_INDEX_REQUIRE = 5  # 是否必填
EXCEL_HEADER_ROW_INDEX_DEPENDS = 6  # 依赖关系
EXCEL_HEADER_ROW_INDEX_TARGET = 7  # 导出目标

EXCEL_ROW_START = 1  # 忽略的行数
EXCEL_COL_START = 1  # 忽略的列数

# ============================ 字段导出目标 =================================
TARGET_CLIENT = "c"  # 导出到客户端
TARGET_SERVER = "s"  # 导出到服务端
TARGET_ALL = "cs"  # 导出到双端
ALL_EXPORT_TARGETS = [TARGET_CLIENT, TARGET_SERVER, TARGET_ALL]
# ============================ 字段导出目标 =================================

# ============================ 字段类型 =================================
VALUE_TYPE_INT = "int"  # 整数
VALUE_TYPE_FLOAT = "float"  # 浮点数
VALUE_TYPE_STR = "string"  # 字符串
VALUE_TYPE_BOOL = "bool"  # bool值
VALUE_TYPE_TABLE = "table"  # 字典
ALL_VALUE_TYPES = [VALUE_TYPE_INT, VALUE_TYPE_FLOAT, VALUE_TYPE_STR, VALUE_TYPE_BOOL, VALUE_TYPE_TABLE]  # 所有的字段类型
KEY_VALUE_TYPES = [VALUE_TYPE_INT, VALUE_TYPE_FLOAT, VALUE_TYPE_STR, VALUE_TYPE_BOOL]  # 可以作为主键的字段类型
BOOL_TYPE_TRUE_VALUES = ["T", "TRUE"]  # bool类型为true的值
# ============================ 字段类型 =================================

PATTERN_LETTER_NAME = r"^[a-zA-Z_][a-zA-Z0-9_]*$"  # 匹配字段英文名

# 导出结果目录

TEMP_DIR = "./.export_table_temp"  # 导出结果临时目录
TEMP_CLIENT_DIR = TEMP_DIR + "/client"
TEMP_SERVER_DIR = TEMP_DIR + "/server"

# 导出的数据是否需要放到table里
TEXT_MODEL_WITHOUT_DICT = 0  # 数据外面不加大括号
TEXT_MODEL_WITHIN_DICT = 1  # 数据放到大括号里

FLOAT_DECIMAL_MIN = 0  # 浮点数最小保留小数位数
FLOAT_DECIMAL_MAX = 5  # 浮点数最大保留小数位数
FLOAT_DECIMAL_DEFAULT = 2  # 浮点数小数位默认值

CLIENT_BEFORE_PROCESS = 1 << 0  # 客户端表格预处理
CLIENT_AFTER_PROCESS = 1 << 1  # 客户端表格后处理
SERVER_BEFORE_PROCESS = 1 << 2  # 服务端表格预处理
SERVER_AFTER_PROCESS = 1 << 3  # 服务端表格后处理

# 写文件方式
WRITE_TYPE_DICT = 0  # 按key-value的形式,默认行为
WRITE_TYPE_NO_VALUE_NAME = 1  # 隐藏value的名字部分

# 依赖关系
DEPEND_NAME_REF = "_ref"  # 单个字段依赖关系
DEPEND_NAME_KEY_REF = "_key_ref"  # table的key依赖关系, 如果是数组则是数组元素的依赖关系
