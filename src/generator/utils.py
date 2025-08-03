import re

from jinja2 import Environment, FileSystemLoader

_BUILTIN_TYPES = {
    "str",
    "number",
    "int",
    "int8",
    "int16",
    "int32",
    "int64",
    "uint",
    "uint8",
    "uint16",
    "uint32",
    "uint64",
    "size",
    "bool",
    "any",
    "null",
    "q_empty",
}

_BUILTIN_TO_GO = {
    "str": "string",
    "number": "float64",
    "int": "int",
    "int8": "int8",
    "int16": "int16",
    "int32": "int32",
    "int64": "int64",
    "uint": "uint",
    "uint8": "uint8",
    "uint16": "uint16",
    "uint32": "uint32",
    "uint64": "uint64",
    "size": "uint64",
    "bool": "bool",
    "any": "interface{}",
    "null": "Null",
    "q_empty": "QEmpty",
}


def is_builtin_type(name: str) -> bool:
    if name in _BUILTIN_TO_GO:
        return True
    return False


def builtin_to_go(name: str) -> str:
    if name in _BUILTIN_TO_GO:
        return _BUILTIN_TO_GO[name]

    return name


def capitalize(s: str) -> str:
    return s[0].upper() + s[1:] if s else s


def uncapitalize(s):
    return s[0].lower() + s[1:] if s else s


def to_go_camel_case(s: str) -> str:
    # Remove invalid characters and split into parts
    parts = re.split(r"[^a-zA-Z0-9]+", s)

    # Filter out empty strings and convert to camelCase
    if not parts:
        return "x"  # fallback for empty input

    result = "".join([capitalize(p) for p in parts if p])

    # Ensure first character is a letter (prepend 'x' if needed)
    if not result or not result[0].isalpha():
        result = "x" + result

    return result


def get_environment(path) -> Environment:
    env = Environment(
        loader=FileSystemLoader(path), extensions=["jinja2.ext.loopcontrols"]
    )

    env.globals["to_go_camel_case"] = to_go_camel_case
    env.globals["capitalize"] = capitalize
    env.globals["uncapitalize"] = uncapitalize
    env.globals["builtin_to_go"] = builtin_to_go
    env.globals["is_builtin_type"] = is_builtin_type

    return env
