# Copyright 2020 ICON Foundation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

__all__ = "get_score_api"

from inspect import Signature, Parameter
from typing import List, Dict, Mapping, Iterable, Any, Union

from . import (
    get_args,
    get_origin,
    is_base_type,
    is_struct,
)
from .element import (
    ScoreElementMetadata,
    FunctionMetadata,
    EventLogMetadata,
)
from ..icon_score_constant import STR_FALLBACK, ScoreFlag
from ...base.exception import IllegalFormatException, InvalidParamsException
from ...ipc import APIType, APIFlag

APIFlagsMask = APIFlag.READONLY \
               | APIFlag.EXTERNAL \
               | APIFlag.PAYABLE

"""Utils to support icx_getScoreApi method
"""


def get_score_api(elements: Iterable[ScoreElementMetadata]) -> List:
    """Returns score api used in icx_getScoreApi JSON-RPC method

    :param elements:
    :return:
    """

    api = []

    for element in elements:
        if isinstance(element, FunctionMetadata):
            func: FunctionMetadata = element
            if func.flag == ScoreFlag.PAYABLE:
                continue

            item = _get_function(func.name, func.flag, func.signature)
        elif isinstance(element, EventLogMetadata):
            eventlog: EventLogMetadata = element
            item = _get_eventlog(eventlog.name, eventlog.signature, eventlog.indexed_args_count)
        else:
            raise IllegalFormatException(f"Invalid score element: {element} {type(element)}")

        api.append(item)

    return api


def _get_function(func_name: str, flag: ScoreFlag, sig: Signature) -> List:
    info = []
    if _is_fallback(func_name, sig, flag & ScoreFlag.PAYABLE):
        info.append(APIType.FALLBACK)
    else:
        info.append(APIType.FUNCTION)
    info.append(func_name)
    info.append(_convert_to_proxy_flag(flag))
    info.append(_get_inputs(sig.parameters))
    if flag & ScoreFlag.READONLY:
        info.append(_get_outputs(sig.return_annotation))
    else:
        info.append([])
    return info


def _get_eventlog(func_name: str, sig: Signature, indexed_args_count: int) -> List:
    params = sig.parameters

    inputs = []
    for name, param in params.items():
        annotation = param.annotation
        type_hint = str if annotation is Parameter.empty else annotation
        inp: Dict = _get_input(name, type_hint, param.default)
        if len(inputs) < indexed_args_count:
            inp["indexed"] = True
        inputs.append(inp)

    return [
        APIType.EVENT,
        func_name,
        indexed_args_count,
        inputs
    ]


def _convert_to_proxy_flag(score_flag: ScoreFlag):
    flags = score_flag.value & APIFlagsMask
    if score_flag & ScoreFlag.ISOLATED:
        flags |= APIFlag.ISOLATED
    return flags


def _is_fallback(func_name: str, sig: Signature, is_payable: bool) -> bool:
    ret: bool = func_name == STR_FALLBACK and is_payable
    if ret:
        if len(sig.parameters) > 1:
            raise InvalidParamsException("Invalid fallback signature")

        return_annotation = sig.return_annotation
        if return_annotation not in (None, Signature.empty):
            raise InvalidParamsException("Invalid fallback signature")

    return ret


def _get_inputs(params: Mapping[str, Parameter]) -> list:
    inputs = []

    for name, param in params.items():
        if param.kind == Parameter.VAR_KEYWORD:
            continue

        annotation = param.annotation
        type_hint = str if annotation is Parameter.empty else annotation

        inputs.append(_get_input(name, type_hint, param.default))

    return inputs


def _get_input(name: str, type_hint: type, default: Any) -> Dict:
    inp = {"name": name}

    # Add default parameter value to score api
    if default is not Parameter.empty:
        if default is not None and not isinstance(default, type_hint):
            raise InvalidParamsException(
                f"Default params type mismatch. value: {default} type: {type_hint}")

        inp["default"] = default

    type_hints: List[type] = _split_type_hint(type_hint)
    inp["type"] = _type_hints_to_name(type_hints)

    last_type_hint: type = type_hints[-1]

    if is_struct(last_type_hint):
        inp["fields"] = _get_fields(last_type_hint)

    return inp


def _split_type_hint(type_hint: type) -> List[type]:
    type_hints = [type_hint]
    ret = []

    while len(type_hints) > 0:
        type_hint = type_hints.pop(0)
        origin: type = get_origin(type_hint)
        ret.append(origin)

        if origin is list:
            args = get_args(type_hint)
            if len(args) != 1:
                raise IllegalFormatException(f"Invalid type: {type_hint}")

            type_hints.append(args[0])
        elif origin is Union:
            args = get_args(type_hint)
            if not (len(args) == 2 and args[1] is type(None)):
                raise IllegalFormatException(f"Invalid type: {type_hint}")

            type_hints.append(args[0])

    return ret


def _type_hints_to_name(type_hints: List[type]) -> str:
    def func():
        for _type in type_hints:
            if _type is Union:
                continue

            if _type is list:
                yield "[]"
            elif is_base_type(_type):
                yield _type.__name__
            elif is_struct(_type):
                yield "struct"

    return "".join(func())


def _get_fields(struct: type) -> List[dict]:
    """Returns fields info from struct

    :param struct: struct type
    :return:
    """
    # annotations is a dictionary containing key-type pair
    # which has field_name as a key and type as a value
    annotations = struct.__annotations__

    fields = []
    for name, type_hint in annotations.items():
        field = {"name": name}

        type_hints: List[type] = _split_type_hint(type_hint)
        field["type"] = _type_hints_to_name(type_hints)

        last_type_hint: type = type_hints[-1]
        if is_struct(last_type_hint):
            field["fields"] = _get_fields(last_type_hint)

        fields.append(field)

    return fields


def _get_outputs(type_hint: type) -> List:
    origin = get_origin(type_hint)

    if is_base_type(origin):
        type_name = origin.__name__
    elif is_struct(origin) or origin is dict:
        type_name = "dict"
    elif origin is list:
        type_name = "list"
    else:
        return []

    return [{"type": type_name}]
