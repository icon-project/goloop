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

from collections import OrderedDict
from collections.abc import MutableMapping
from inspect import Parameter, Signature, getmembers, isfunction, signature
from typing import Any, List, Mapping, Set, Union

from . import (
    get_annotations,
    get_args,
    get_origin,
    is_base_type,
    is_struct,
    isinstance_ex,
    name_to_type,
)
from ..icon_score_constant import (
    CONST_INDEXED_ARGS_COUNT,
    CONST_SCORE_FLAG,
    STR_ON_INSTALL,
    STR_ON_UPDATE,
    ScoreFlag,
)
from ... import utils
from ...base.exception import (
    IllegalFormatException,
    InvalidInstanceException,
    InvalidParamsException,
)

_VALID_SCORE_FLAG_COMBINATION = {
    ScoreFlag.EXTERNAL,
    ScoreFlag.EXTERNAL | ScoreFlag.PAYABLE,
    ScoreFlag.EXTERNAL | ScoreFlag.READONLY,
    ScoreFlag.FALLBACK | ScoreFlag.PAYABLE,
    ScoreFlag.EVENTLOG,
    ScoreFlag.INTERFACE,
}

_MAX_STRUCT_FIELDS = 16


def normalize_signature(func: callable) -> Signature:
    """Normalize signature of score methods

    1. Normalize type hint: ex) no type hint -> str
    2. Remove "self" parameter

    :param func: function attribute from class
    :return:
    """
    sig = signature(func)
    parameters = sig.parameters
    new_params = []

    normalized = False

    # CAUTION:
    # def A:
    #     def func(self):
    #         pass
    #
    #     @classmethod
    #     def cfunc(self):
    #         pass
    #
    # inspect.isfunction(A.func) == True
    # inspect.isfunction(A().func) == False
    # inspect.ismethod(A.func) == False
    # inspect.ismethod(A().func) == True
    # inspect.isfunction(A.cfunc) == False
    # inspect.ismethod(A.cfunc) == True
    # inspect.isfunction(A().cfunc) == False
    # inspect.ismethod(A().cfunc) == True
    is_regular_method: bool = isfunction(func)

    for i, k in enumerate(parameters):
        # Remove "self" parameter from signature of regular method
        if i == 0 and k == "self" and is_regular_method:
            new_param = None
        else:
            new_param = normalize_parameter(parameters[k])
            new_params.append(new_param)

        if new_param is not parameters[k]:
            normalized = True

    return_annotation = normalize_return_annotation(sig.return_annotation)
    if return_annotation is not sig.return_annotation:
        normalized = True

    if normalized:
        sig = sig.replace(parameters=new_params, return_annotation=return_annotation)

    return sig


def normalize_parameter(parameter: Parameter) -> Parameter:
    if parameter.kind != Parameter.POSITIONAL_OR_KEYWORD:
        raise IllegalFormatException(
            f"Invalid signature: name={parameter.name} kind={parameter.kind}")

    annotation = parameter.annotation

    if annotation == Parameter.empty:
        type_hint = str
    else:
        type_hint = normalize_type_hint(annotation)

    # a: int = None -> a: Union[int, None] = None
    if parameter.default is None and get_origin(type_hint) is not Union:
        type_hint = Union[type_hint, None]

    check_parameter_default_type(type_hint, parameter.default)

    if type_hint == annotation:
        # Nothing to update
        return parameter

    return parameter.replace(annotation=type_hint)


def normalize_return_annotation(return_annotation: type) -> Union[type, Signature.empty]:
    if return_annotation in (None, Signature.empty):
        return Signature.empty

    return return_annotation


def check_parameter_default_type(type_hint: type, default: Any):
    # default value type check
    if default in (Parameter.empty, None):
        return

    origin = get_origin(type_hint)

    if origin is Union:
        default_type = get_args(type_hint)[0]
    else:
        default_type = origin

    if not isinstance_ex(default, default_type):
        raise InvalidParamsException(
            f'Default value type mismatch. value={default} type={type_hint}')


def normalize_type_hint(type_hint) -> type:
    # If type hint is str, convert it to type hint
    if isinstance(type_hint, str):
        if type_hint == "Address":
            type_hint = name_to_type(type_hint)
        else:
            raise IllegalFormatException(f"Invalid type hint: {repr(type_hint)}")

    origin = get_origin(type_hint)

    if is_base_type(origin):
        return type_hint

    if is_struct(origin):
        # Check if type_hint cycling or invalid nested type hint are present
        if not is_struct_valid(origin):
            raise IllegalFormatException(f"Invalid type hint: {type_hint}")
        return type_hint

    if origin is list:
        return normalize_list_type_hint(type_hint)
    elif origin is dict:
        return normalize_dict_type_hint(type_hint)
    elif origin is Union:
        return normalize_union_type_hint(type_hint)

    raise IllegalFormatException(f"Unsupported type hint: {type_hint}")


def normalize_list_type_hint(type_hint: type) -> type:
    args = get_args(type_hint)

    if len(args) == 1:
        return List[normalize_type_hint(args[0])]

    raise IllegalFormatException(f"Invalid type hint: {type_hint}")


def normalize_dict_type_hint(type_hint: type) -> type:
    raise IllegalFormatException(f"Dict not supported: {type_hint}")

    # TODO: The below codes are commented out to prevent Dict type for being used as SCORE parameter
    # args = get_args(type_hint)
    #
    # if len(args) == 2 and args[0] is str:
    #     return Dict[str, normalize_type_hint(args[1])]
    #
    # raise IllegalFormatException(f"Invalid type hint: {type_hint}")


def normalize_union_type_hint(type_hint: type) -> type:
    args = get_args(type_hint)

    if len(args) == 2 and type(None) in args:
        arg = args[0] if args[1] is type(None) else args[1]
        if arg is not None and arg:
            return Union[normalize_type_hint(arg), None]

    raise IllegalFormatException(f"Invalid type hint: {type_hint}")


def verify_score_flag(flag: ScoreFlag, allow_payable_only: bool = True):
    """Check if score flag combination is valid

    If the flag combination is not valid, raise an exception
    """
    if flag in _VALID_SCORE_FLAG_COMBINATION:
        return

    if allow_payable_only and flag == ScoreFlag.PAYABLE:
        # Allow BH-6573883 to be synchronized for mainnet backward compatibility
        return

    raise IllegalFormatException(f"Invalid score decorator: {flag}")


class ScoreElementMetadata(object):
    def __init__(self, element: callable):
        self._signature: Signature = normalize_signature(element)
        self._element = element

    @property
    def element(self) -> callable:
        return self._element

    @property
    def name(self) -> str:
        return self._element.__name__

    @property
    def flag(self) -> ScoreFlag:
        return get_score_flag(self._element)

    @property
    def signature(self) -> Signature:
        return self._signature


class FunctionMetadata(ScoreElementMetadata):
    """Represents metadata of an exposed function in a SCORE

    """
    def __init__(self, func: callable):
        super().__init__(func)
        self._verify()

    @property
    def is_external(self) -> bool:
        return bool(self.flag & ScoreFlag.EXTERNAL)

    @property
    def is_payable(self) -> bool:
        return bool(self.flag & ScoreFlag.PAYABLE)

    @property
    def is_readonly(self) -> bool:
        return bool(self.flag & ScoreFlag.READONLY)

    @property
    def is_fallback(self) -> bool:
        return utils.is_all_flag_on(
            self.flag, ScoreFlag.FALLBACK | ScoreFlag.PAYABLE)

    def _verify(self):
        if self.is_fallback:
            self._verify_fallback_signature()

    def _verify_fallback_signature(self):
        """Verify if the signature of fallback() is valid

        fallback function must have no parameters
        """
        sig = self.signature

        if not (
                len(sig.parameters) == 0
                and sig.return_annotation in (None, Signature.empty)
        ):
            raise IllegalFormatException("Invalid fallback signature")


class EventLogMetadata(ScoreElementMetadata):
    """Represents metadata of an eventlog declared in a SCORE
    """

    def __init__(self, eventlog: callable):
        super().__init__(eventlog)

    @property
    def indexed_args_count(self) -> int:
        return getattr(self.element, CONST_INDEXED_ARGS_COUNT, 0)


class ScoreElementMetadataContainer(MutableMapping):
    """Container which has score elements like function and eventlog
    """

    def __init__(self):
        self._elements = OrderedDict()
        self._externals = 0
        self._eventlogs = 0
        self._readonly = False

    @property
    def externals(self) -> int:
        return self._externals

    @property
    def eventlogs(self) -> int:
        return self._eventlogs

    def __getitem__(self, k: str) -> ScoreElementMetadata:
        return self._elements[k]

    def __setitem__(self, k: str, v: ScoreElementMetadata) -> None:
        self._check_writable()
        self._elements[k] = v
        if k in (STR_ON_INSTALL, STR_ON_UPDATE):
            return

        if isinstance(v, FunctionMetadata):
            self._externals += 1
        elif isinstance(v, EventLogMetadata):
            self._eventlogs += 1
        else:
            raise InvalidInstanceException(f"Invalid element: {v}")

    def __iter__(self):
        for k in self._elements:
            yield k

    def __len__(self) -> int:
        return len(self._elements)

    def __delitem__(self, k: str) -> None:
        self._check_writable()

        element = self._elements[k]
        del self._elements[k]

        if is_any_score_flag_on(element, ScoreFlag.EVENTLOG):
            self._eventlogs -= 1
        else:
            self._externals -= 1

    def _check_writable(self):
        if self._readonly:
            raise InvalidInstanceException(f"{self.__class__.__name__} not writable")

    def freeze(self):
        self._readonly = True


def create_score_element_metadata(cls: type) -> Mapping:
    elements = ScoreElementMetadataContainer()

    for name, func in getmembers(cls, predicate=isfunction):
        if name.startswith("__"):
            continue

        flag = get_score_flag(func)

        # goloop needs to have these init functions explicitly
        if name == STR_ON_INSTALL or name == STR_ON_UPDATE:
            if hasattr(func, "__isabstractmethod__"):
                continue
            if utils.is_any_flag_on(flag, ScoreFlag.ALL):
                raise IllegalFormatException(f"Invalid decorators in {name}")
            elements[name] = FunctionMetadata(func)
            continue

        # Collect the only functions with the following flags
        if utils.is_any_flag_on(flag, ScoreFlag.FUNC | ScoreFlag.EVENTLOG):
            verify_score_flag(flag)
            elements[name] = __get_score_element_metadata(func)

    elements.freeze()
    return elements


def __get_score_element_metadata(element: callable) -> Union[FunctionMetadata, EventLogMetadata]:
    flags = get_score_flag(element)

    if flags & ScoreFlag.EVENTLOG:
        return EventLogMetadata(element)
    else:
        return FunctionMetadata(element)


def get_score_flag(obj: callable, default: ScoreFlag = ScoreFlag.NONE) -> ScoreFlag:
    return getattr(obj, CONST_SCORE_FLAG, default)


def set_score_flag(obj: callable, flag: ScoreFlag) -> ScoreFlag:
    setattr(obj, CONST_SCORE_FLAG, flag)
    return flag


def set_score_flag_on(obj: callable, flag: ScoreFlag) -> ScoreFlag:
    flag |= get_score_flag(obj)
    set_score_flag(obj, flag)
    return flag


def is_all_score_flag_on(obj: callable, flag: ScoreFlag) -> bool:
    return utils.is_all_flag_on(get_score_flag(obj), flag)


def is_any_score_flag_on(obj: callable, flag: ScoreFlag) -> bool:
    return utils.is_any_flag_on(get_score_flag(obj), flag)


def is_struct_valid(type_hint: type) -> bool:
    try:
        check_if_struct_is_valid(type_hint)
    except:
        return False

    return True


def check_if_struct_is_valid(type_hint: type, structs: Set[Any] = None):
    if structs is None:
        structs = set()

    if type_hint in structs:
        raise IllegalFormatException(f"Circular type hint: {type_hint}")

    structs.add(type_hint)
    annotations = get_annotations(type_hint, None)

    size = len(annotations)
    if size > _MAX_STRUCT_FIELDS:
        raise IllegalFormatException(
            f"Too many fields in struct: {size} > {_MAX_STRUCT_FIELDS}")

    for type_hint in annotations.values():
        origin = get_origin(type_hint)
        if is_base_type(origin):
            continue

        if origin is list:
            normalize_list_type_hint(type_hint)
        elif origin is dict:
            normalize_dict_type_hint(type_hint)
        elif origin is Union:
            normalize_union_type_hint(type_hint)
        elif is_struct(origin):
            check_if_struct_is_valid(type_hint, structs)
        else:
            raise IllegalFormatException(f"Invalid type hint: {type_hint}")
