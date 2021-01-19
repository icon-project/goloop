# Copyright 2018 ICON Foundation
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

from inspect import signature, Signature, Parameter, isclass
from typing import Any, Optional

from .icon_score_constant import (
    BaseType,
    CONST_INDEXED_ARGS_COUNT,
    STR_FALLBACK, STR_ON_INSTALL, STR_ON_UPDATE,
    ScoreFlag,
)
from .typing.element import get_score_flag
from ..base.address import Address
from ..base.exception import IllegalFormatException, InvalidParamsException
from ..utils import get_main_type_from_annotations_type
from ..ipc.proxy import APIType, APIFlag

APIFlagsMask = APIFlag.READONLY \
               | APIFlag.EXTERNAL \
               | APIFlag.PAYABLE


class ScoreApiGenerator:

    __API_TYPE = 'type'
    __API_NAME = 'name'
    __API_INPUTS = 'inputs'
    __API_OUTPUTS = 'outputs'
    __API_PAYABLE = 'payable'
    __API_READONLY = 'readonly'
    __API_INPUTS_INDEXED = 'indexed'
    __API_INPUTS_DEFAULT = 'default'
    __API_PARAMS_ADDRESS = 'Address'
    __API_PARAMS_INDEXED = 'Indexed'
    __API_TYPE_FUNCTION = 'function'
    __API_TYPE_EVENT = 'eventlog'

    __API_TYPE_FALLBACK = STR_FALLBACK
    __API_TYPE_ON_INSTALL = STR_ON_INSTALL
    __API_TYPE_ON_UPDATE = STR_ON_UPDATE

    @staticmethod
    def generate(score_funcs: list) -> list:
        api = []
        ScoreApiGenerator.__generate_functions(api, score_funcs)
        ScoreApiGenerator.__generate_events(api, score_funcs)
        return api

    @staticmethod
    def __generate_functions(src: list, score_funcs: list) -> None:
        for func in score_funcs:
            try:
                score_flag = get_score_flag(func)
                if score_flag & ScoreFlag.EXTERNAL or \
                        func.__name__ == ScoreApiGenerator.__API_TYPE_ON_INSTALL or \
                        func.__name__ == ScoreApiGenerator.__API_TYPE_ON_UPDATE or \
                        func.__name__ == ScoreApiGenerator.__API_TYPE_FALLBACK:
                    src.append(ScoreApiGenerator.__generate_function_info(
                        func.__name__, score_flag, signature(func)))
            except IllegalFormatException as e:
                raise IllegalFormatException(f"{e.message} at {func.__name__}")

    @staticmethod
    def __generate_function_info(func_name: str, score_flag: ScoreFlag, sig_info: 'Signature') -> list:
        is_readonly = bool(score_flag & ScoreFlag.READONLY)
        info = list()
        if func_name == ScoreApiGenerator.__API_TYPE_FALLBACK:
            info.append(APIType.FALLBACK)
        else:
            info.append(APIType.FUNCTION)
        info.append(func_name)
        info.append(ScoreApiGenerator.__convert_to_proxy(score_flag))
        info.append(ScoreApiGenerator.__generate_inputs(dict(sig_info.parameters)))
        info.append(ScoreApiGenerator.__generate_output(sig_info.return_annotation, is_readonly))
        return info

    @staticmethod
    def __convert_to_proxy(score_flag: ScoreFlag):
        flags = score_flag.value & APIFlagsMask
        if score_flag & ScoreFlag.ISOLATED:
            flags |= APIFlag.ISOLATED
        return flags

    @staticmethod
    def __generate_events(src: list, score_funcs: list) -> None:
        event_funcs = {
            func.__name__: signature(func) for func in score_funcs
            if get_score_flag(func) & ScoreFlag.EVENTLOG
        }

        indexed_args_counts = {
            func.__name__: getattr(func, CONST_INDEXED_ARGS_COUNT, 0)
            for func in score_funcs
            if getattr(func, CONST_INDEXED_ARGS_COUNT, 0)
        }

        for func_name, event in event_funcs.items():
            index_args_count = indexed_args_counts.get(func_name, 0)
            src.append(ScoreApiGenerator.__generate_event(func_name, event, index_args_count))

    @staticmethod
    def __generate_event(func_name: str, sig_info: 'Signature', index_args_count: int) -> list:
        info = list()
        info.append(APIType.EVENT)
        info.append(func_name)
        info.append(index_args_count)
        info.append(ScoreApiGenerator.__generate_inputs(dict(sig_info.parameters), index_args_count))
        return info

    @staticmethod
    def __generate_output(params_type: Any, is_readonly: bool) -> list:
        info_list = []

        if not is_readonly:
            return info_list

        if params_type is Signature.empty:
            raise IllegalFormatException(
                "Returning type should be declared in read-only functions")

        main_type = get_main_type_from_annotations_type(params_type)
        main_type = ScoreApiGenerator.__convert_str_to_type(main_type)

        # At first, finds if the type is a 'list' or a 'dict'
        # if not, finds a base type
        find = (t for t in [list, dict]
                if isclass(main_type) and issubclass(main_type, t))
        api_type = next(find, None)
        if api_type is None:
            api_type = ScoreApiGenerator.__find_base_super_type(main_type)
        if api_type is None:
            raise IllegalFormatException(f"Unsupported type for '{params_type}'")

        info = dict()
        info[ScoreApiGenerator.__API_TYPE] = api_type.__name__
        info_list.append(info)
        return info_list

    @staticmethod
    def __convert_str_to_type(params_type: Any) -> Any:
        if not isinstance(params_type, str):
            return params_type

        if params_type == 'Address':
            return Address
        else:
            return params_type

    @staticmethod
    def __generate_inputs(params: dict, index_args_count: int = 0) -> list:
        tmp_list = []
        args_index = 0
        for param_name, param in params.items():
            if param_name == 'self' or param_name == 'cls':
                continue
            is_indexed = args_index < index_args_count
            args_index += 1
            ScoreApiGenerator.__generate_input(tmp_list, param, is_indexed)
        return tmp_list

    @staticmethod
    def __generate_input(src: list, param: 'Parameter', is_indexed: bool):
        # If there's no hint of argument in the function declaration,
        # raise an exception
        if param.annotation is Parameter.empty:
            raise IllegalFormatException(f"Missing argument hint for '{param.name}'")

        main_type = get_main_type_from_annotations_type(param.annotation)
        main_type = ScoreApiGenerator.__convert_str_to_type(main_type)
        api_type = ScoreApiGenerator.__find_base_super_type(main_type)
        if api_type is None:
            raise IllegalFormatException(
                f"Unsupported type for '{param.name}: {param.annotation}'")
        info = dict()
        info[ScoreApiGenerator.__API_NAME] = param.name
        info[ScoreApiGenerator.__API_TYPE] = api_type.__name__
        if is_indexed:
            info[ScoreApiGenerator.__API_INPUTS_INDEXED] = is_indexed
        if param.default is not Parameter.empty:
            if param.default is not None and not isinstance(param.default, main_type):
                raise InvalidParamsException(f'Default params type mismatch. value: {param.default} type: {main_type}')
            # the default param value will be encoded at ipc.proxy
            info[ScoreApiGenerator.__API_INPUTS_DEFAULT] = param.default
        src.append(info)

    @staticmethod
    def __find_base_super_type(t: type) -> Optional[type]:
        """
        Finds a base type of the input and returns it if any
        :param t: target
        :return: base_super_type
        """
        # pylint: disable=no-member
        find = (base_type for base_type in BaseType.__constraints__
                if isclass(t) and issubclass(t, base_type))
        return next(find, None)
