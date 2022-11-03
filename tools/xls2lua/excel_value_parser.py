import const


def _check_int(value):
    int(value)
    return True


def _check_float(value, decimal):
    decimal = min(const.FLOAT_DECIMAL_MAX, max(const.FLOAT_DECIMAL_MIN, decimal))
    format(value, ".{}f".format(decimal))
    return True


def _check_bool(value):
    if value not in const.BOOL_TYPE_TRUE_VALUES:
        if str.lower(value) in ["t", "true"]:
            raise Exception("请填写大写的: {}".format(const.BOOL_TYPE_TRUE_VALUES))
        else:
            raise Exception("bool类型仅支持填: {}".format(const.BOOL_TYPE_TRUE_VALUES))
    return True


# 将excel单元格的值转换为对应类型的值的字符串
def check_excel_value(value, value_type):
    try:
        if value_type == const.VALUE_TYPE_INT:
            return _check_int(value)
        elif value_type == const.VALUE_TYPE_BOOL:
            return _check_bool(value)
        elif value_type == const.VALUE_TYPE_FLOAT:
            return _check_float(value, const.FLOAT_DECIMAL_DEFAULT)  # 未指定小数点位数,默认保留两位
        elif value_type.startswith(const.VALUE_TYPE_FLOAT):
            info = value_type.split(",")
            return _check_float(value, int(info[1]))  # 指定了小数点位数
        else:
            return True
    except Exception as ex:
        raise Exception(ex)


def _to_int(value):
    return int(value)


def _to_string(value):
    return str(value)


def _to_bool(value):
    return value in const.BOOL_TYPE_TRUE_VALUES


def _to_float(value, decimal):
    decimal = min(const.FLOAT_DECIMAL_MAX, max(const.FLOAT_DECIMAL_MIN, decimal))
    v = round(value, decimal)
    return v


def _parse_dict_value(v):
    if ',' in v:
        return [_parse_dict_value(i) for i in v.split(",")]
    else:
        if '"' in v:
            return str(v.strip('"'))
        try:
            return int(v)
        except:
            try:
                return float(v)
            except:
                return str(v)


def _to_dict(value):
    if ":" in value:
        d = {}
        if ";" in value:
            arr = value.split(";")
        elif "," in value:
            arr = value.split(",")
        else:
            arr = [value]
        for item in arr:
            if not item:
                continue
            ta = item.split(":")
            if len(ta) == 1:
                if not isinstance(d, list):
                    temp = []
                    for k, v in d.items():
                        temp.append({k: v})
                    temp.append(_parse_dict_value(ta[0]))
                    d = temp
                else:
                    d.append(_parse_dict_value(ta[0]))
            else:
                if not isinstance(d, list):
                    d[_parse_dict_value(ta[0])] = _parse_dict_value(ta[1])
                else:
                    d.append({_parse_dict_value(ta[0]): _parse_dict_value(ta[1])})
        return d
    elif ";" in value:
        d = [_parse_dict_value(v) for v in value.split(";")]
        return d
    elif "," in value:
        d = [_parse_dict_value(v) for v in value.split(",")]
        return d
    else:
        return {}


# 将excel的单元格数据转换为python数据
def convert_excel_value_to_python(value, value_type):
    try:
        if value_type == const.VALUE_TYPE_INT:
            return _to_int(value)
        elif value_type == const.VALUE_TYPE_STR:
            return _to_string(value)
        elif value_type == const.VALUE_TYPE_BOOL:
            return _to_bool(value)
        elif value_type == const.VALUE_TYPE_TABLE:
            return _to_dict(value)
        elif value_type == const.VALUE_TYPE_FLOAT:
            return _to_float(value, const.FLOAT_DECIMAL_DEFAULT)
        elif value_type.startswith(const.VALUE_TYPE_FLOAT):
            info = value_type.split(",")
            return _to_float(value, int(info[1]))
    except Exception as ex:
        print(f"convert value: {value}, value_type: {value_type} failed")
        raise ex
