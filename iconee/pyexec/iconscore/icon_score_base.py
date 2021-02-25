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

import os
import warnings
from abc import abstractmethod, ABC, ABCMeta
from functools import partial, wraps
from inspect import Parameter, isfunction, signature
from typing import Callable, Any, List, Mapping, Tuple

from .icon_score_base2 import (
    Block, Icx, InterfaceScore,
    create_interface_score, revert,
)
from .icon_score_constant import (
    BaseType, T,
    CONST_CLASS_API,
    CONST_CLASS_ELEMENT_METADATA,
    CONST_INDEXED_ARGS_COUNT,
    FORMAT_DECORATOR_DUPLICATED,
    FORMAT_IS_NOT_DERIVED_OF_OBJECT,
    FORMAT_IS_NOT_FUNCTION_OBJECT,
    STR_FALLBACK, STR_ON_INSTALL, STR_ON_UPDATE,
    ScoreFlag,
)
from .icon_score_context import ContextGetter, ContextContainer
from .icon_score_eventlog import EventLogEmitter
from .internal_call import InternalCall, ChainScore
from .typing.definition import get_score_api
from .typing.element import (
    FunctionMetadata,
    ScoreElementMetadata,
    ScoreElementMetadataContainer,
    create_score_element_metadata,
    is_any_score_flag_on,
    set_score_flag_on,
)
from ..base.address import Address
from ..base.exception import *
from ..base.message import Message
from ..base.transaction import Transaction
from ..database.db import IconScoreDatabase
from ..icon_constant import ICX_TRANSFER_EVENT_LOG, IconScoreContextType, Revision
from ..utils import get_main_type_from_annotations_type

INDEXED_ARGS_LIMIT = 3


def interface(func):
    """
    A decorator for the functions of InterfaceScore.

    If other SCORE has the function whose signature is the same as defined with @interface decorator,
    the function can be invoked via InterfaceScore class instance
    """

    cls_name, func_name = str(func.__qualname__).split('.')
    if not isfunction(func):
        raise IllegalFormatException(FORMAT_IS_NOT_FUNCTION_OBJECT.format(func, cls_name))

    if is_any_score_flag_on(func, ScoreFlag.INTERFACE):
        raise InvalidInterfaceException(FORMAT_DECORATOR_DUPLICATED.format('interface', func_name, cls_name))

    set_score_flag_on(func, ScoreFlag.INTERFACE)

    @wraps(func)
    def __wrapper(calling_obj: "InterfaceScore", *args, **kwargs):
        if not isinstance(calling_obj, InterfaceScore):
            raise InvalidInstanceException(
                FORMAT_IS_NOT_DERIVED_OF_OBJECT.format(InterfaceScore.__name__))

        context = ContextContainer._get_context()
        addr_to = calling_obj.addr_to
        addr_from: 'Address' = context.to

        amount: int = getattr(calling_obj, "_InterfaceScore__get_icx")()
        getattr(calling_obj, "_InterfaceScore__reset_icx")()

        if addr_to is None:
            raise InvalidInterfaceException('Cannot create an interface SCORE with a None address')

        return InternalCall.message_call(context, addr_from, addr_to, amount, func_name, args, kwargs)

    return __wrapper


def eventlog(func=None, *, indexed=0):
    """
    Functions with `@eventlog` decorator will include logs in its TxResult as ‘eventlogs’.
    If indexed parameter is set in the decorator, designated number of parameters in the order of
    declaration will be indexed and included in the Bloom filter.
    Indexed parameters and non-indexed parameters are separately stored in TxResult.
    Possible data types for function parameters are primitive types (int, str, bytes, bool, Address).

    It is recommended to declare a function without implementation body.
    Even if the function has a body, it does not be executed.
    When declaring a function, type hinting is a must. Without type hinting, transaction will fail.
    The default value for the parameter can be set.
    At most 3 parameters can be indexed, And index can’t exceed the number of parameters(will raise an error).

    :param indexed: the number of indexed parameters count(maximum 3)
    """
    if func is None:
        return partial(eventlog, indexed=indexed)

    cls_name, func_name = str(func.__qualname__).split('.')
    if not isfunction(func):
        raise IllegalFormatException(FORMAT_IS_NOT_FUNCTION_OBJECT.format(func, cls_name))

    if not list(signature(func).parameters.keys())[0] == 'self':
        raise InvalidEventLogException("'self' is not declared as the first parameter")
    if indexed > INDEXED_ARGS_LIMIT:
        raise InvalidEventLogException(
            f'Indexed arguments overflow: limit={INDEXED_ARGS_LIMIT}')

    parameters = signature(func).parameters.values()
    if len(parameters) - 1 < indexed:
        raise InvalidEventLogException("Index exceeds the number of parameters")

    if is_any_score_flag_on(func, ScoreFlag.EVENTLOG):
        raise InvalidEventLogException(FORMAT_DECORATOR_DUPLICATED.format('eventlog', func_name, cls_name))

    set_score_flag_on(func, ScoreFlag.EVENTLOG)
    setattr(func, CONST_INDEXED_ARGS_COUNT, indexed)
    event_signature = __retrieve_event_signature(func_name, parameters)

    @wraps(func)
    def __wrapper(calling_obj: 'IconScoreBase', *args, **kwargs):
        if not (isinstance(calling_obj, IconScoreBase)):
            raise InvalidInstanceException(
                FORMAT_IS_NOT_DERIVED_OF_OBJECT.format(IconScoreBase.__name__))
        try:
            arguments = __resolve_arguments(func_name, parameters, args, kwargs)
        except IllegalFormatException as e:
            raise InvalidEventLogException(e.message)

        if event_signature == ICX_TRANSFER_EVENT_LOG:
            # 'ICXTransfer(Address,Address,int)' is reserved
            raise InvalidEventLogException(
                f'The event log \'{ICX_TRANSFER_EVENT_LOG}\' is reserved')

        return EventLogEmitter.emit_event_log(
            calling_obj._context, calling_obj.address, event_signature, arguments, indexed)

    return __wrapper


def __retrieve_event_signature(function_name, parameters) -> str:
    """
    Retrieves a event signature from the function name and parameters
    :param function_name: name of event function
    :param parameters: Arguments description of the function declaration
    :return: event signature
    """
    type_names: List[str] = []
    for i, param in enumerate(parameters):
        if i > 0:
            # If there's no hint of argument in the function declaration,
            # raise an exception
            if param.annotation is Parameter.empty:
                raise IllegalFormatException(
                    f"Missing argument hint for '{function_name}': '{param.name}'")

            main_type = None
            if isinstance(param.annotation, type):
                main_type = param.annotation
            elif param.annotation == 'Address':
                main_type = Address

            # Raises an exception if the types are not supported
            # pylint: disable=no-member
            if main_type is None or not issubclass(main_type, BaseType.__constraints__):
                raise IllegalFormatException(
                    f"Unsupported type for '{param.name}: {param.annotation}'")

            type_names.append(str(main_type.__name__))
    return f"{function_name}({','.join(type_names)})"


def __resolve_arguments(function_name, parameters, args, kwargs) -> List[Any]:
    """
    Resolves arguments with keeping order as the function declaration
    :param parameters: Arguments description of the function declaration
    :param args: input ordered arguments
    :param kwargs: input keyword arguments
    :return: an ordered list of arguments
    """
    arguments = []
    if len(parameters) - 1 < len(args) + len(kwargs):
        raise IllegalFormatException(
            f"The maximum number of arguments which event log method({function_name}) can accept is exceeded")

    for i, parameter in enumerate(parameters, -1):
        if i < 0:
            # pass the self parameter
            continue
        name = parameter.name
        annotation = parameter.annotation
        if i < len(args):
            # the argument is in the ordered args
            value = args[i]
            if name in kwargs:
                raise IllegalFormatException(
                    f"Duplicated argument value for '{function_name}': {name}")
        else:
            # If arg is over, the argument should be searched on kwargs
            try:
                value = kwargs[name]
            except KeyError:
                if not parameter.default == Parameter.empty:
                    value = parameter.default
                else:
                    raise IllegalFormatException(
                        f"Missing argument value for '{function_name}': {name}")

        main_type = get_main_type_from_annotations_type(annotation)

        if main_type == 'Address':
            main_type = Address

        if value is not None and not isinstance(value, main_type):
            raise IllegalFormatException(
                f"Type mismatch of '{name}': {type(value)}, expected: {main_type}")
        arguments.append(value)
    return arguments


def external(func=None, *, readonly=False):
    """
    A decorator for the function whether the function exposes externally.
    If declared to the function, EOA or another SCORE can call it.
    These functions are registered on the exportable API list.
    Any attempt to call a non-external function from outside the contract will fail.

    If a function is decorated with ‘readonly’ parameters, i.e., `@external(readonly=True)`,
    the function will have read-only access to the state DB. This is similar to view keyword in Solidity.
    If the read-only external function is also decorated with `@payable`, the function call will fail.
    Duplicate declaration of @external will raise an exception on import time.

    :param readonly: True if the function have read-only access to the state DB.
    """
    if func is None:
        return partial(external, readonly=readonly)

    cls_name, func_name = str(func.__qualname__).split('.')
    if not isfunction(func):
        raise IllegalFormatException(FORMAT_IS_NOT_FUNCTION_OBJECT.format(func, cls_name))

    if func_name == STR_FALLBACK:
        raise InvalidExternalException(f"{func_name} cannot be declared as external")

    if is_any_score_flag_on(func, ScoreFlag.EXTERNAL):
        raise InvalidExternalException(FORMAT_DECORATOR_DUPLICATED.format('external', func_name, cls_name))

    flag = ScoreFlag.EXTERNAL
    if readonly:
        flag |= ScoreFlag.READONLY
    set_score_flag_on(func, flag)

    @wraps(func)
    def __wrapper(calling_obj: Any, *args, **kwargs):
        if not (isinstance(calling_obj, IconScoreBase)):
            raise InvalidInstanceException(
                FORMAT_IS_NOT_DERIVED_OF_OBJECT.format(IconScoreBase.__name__))
        res = func(calling_obj, *args, **kwargs)
        return res

    return __wrapper


def payable(func):
    """
    A decorator for the external function.

    If the decorator is declared to the external function,
    it can receive the ICXs and process further works for it.
    If ICXs (msg.value) are passed to a non-payable function, that transaction will fail.
    """
    cls_name, func_name = str(func.__qualname__).split('.')
    if not isfunction(func):
        raise IllegalFormatException(FORMAT_IS_NOT_FUNCTION_OBJECT.format(func, cls_name))

    if is_any_score_flag_on(func, ScoreFlag.PAYABLE):
        raise InvalidPayableException(FORMAT_DECORATOR_DUPLICATED.format('payable', func_name, cls_name))

    flag = ScoreFlag.PAYABLE
    if func_name == STR_FALLBACK:
        # If a function has payable decorator and its name is "fallback",
        # then it is a fallback function
        flag |= ScoreFlag.FALLBACK
    set_score_flag_on(func, flag)

    @wraps(func)
    def __wrapper(calling_obj: Any, *args, **kwargs):
        if not (isinstance(calling_obj, IconScoreBase)):
            raise InvalidInstanceException(
                FORMAT_IS_NOT_DERIVED_OF_OBJECT.format(IconScoreBase.__name__))
        res = func(calling_obj, *args, **kwargs)
        return res

    return __wrapper


def isolated(func):
    cls_name, func_name = str(func.__qualname__).split('.')
    if not isfunction(func):
        raise IllegalFormatException(FORMAT_IS_NOT_FUNCTION_OBJECT.format(func, cls_name))

    if is_any_score_flag_on(func, ScoreFlag.ISOLATED):
        raise IllegalFormatException(FORMAT_DECORATOR_DUPLICATED.format('isolated', func_name, cls_name))

    set_score_flag_on(func, ScoreFlag.ISOLATED)

    @wraps(func)
    def __wrapper(calling_obj: Any, *args, **kwargs):
        if not (isinstance(calling_obj, IconScoreBase)):
            raise InvalidInstanceException(
                FORMAT_IS_NOT_DERIVED_OF_OBJECT.format(IconScoreBase.__name__))
        res = func(calling_obj, *args, **kwargs)
        return res

    return __wrapper


class IconScoreObject(ABC):

    def __init__(self, *args, **kwargs) -> None:
        pass

    def on_install(self, **kwargs) -> None:
        pass

    def on_update(self, **kwargs) -> None:
        pass


class IconScoreBaseMeta(ABCMeta):

    def __new__(mcs, name, bases, namespace, **kwargs):
        cls = super().__new__(mcs, name, bases, namespace, **kwargs)

        if IconScoreObject in bases or name == "IconSystemScoreBase":
            return cls

        if not isinstance(namespace, dict):
            raise InvalidParamsException('namespace is not dict!')

        elements: Mapping[str, ScoreElementMetadata] = create_score_element_metadata(cls)
        setattr(cls, CONST_CLASS_ELEMENT_METADATA, elements)

        # Generate SCORE API list
        api_list = get_score_api(elements.values())
        setattr(cls, CONST_CLASS_API, api_list)

        return cls


class IconScoreBase(IconScoreObject, ContextGetter,
                    metaclass=IconScoreBaseMeta):
    """
    A base class of SCOREs. This class provides facilities and environments to SCORE to run.
    """

    @abstractmethod
    def on_install(self, **kwargs) -> None:
        """
        Invoked when the contract is deployed for the first time,
        and will not be called again on contract update or deletion afterward.
        This is the place where you initialize the state DB.
        """
        super().on_install(**kwargs)

    @abstractmethod
    def on_update(self, **kwargs) -> None:
        """
        Invoked when the contract is deployed for update.
        This is the place where you migrate old states.
        """
        super().on_update(**kwargs)

    @abstractmethod
    def __init__(self, db: 'IconScoreDatabase') -> None:
        """
        A Python init function. Invoked when the contract is loaded at each node.
        Do not put state-changing works in here.
        """
        super().__init__(db)
        self.__db = db
        self.__address = db.address
        self.__owner = self.get_owner(self.__address)
        self.__icx = None

        elements: ScoreElementMetadataContainer = self.__get_score_element_metadata()
        if elements.externals == 0:
            raise InvalidExternalException('There is no external method in the SCORE')

    def fallback(self) -> None:
        """
        fallback function can not be decorated with `@external`.
        (i.e., fallback function is not allowed to be called by external contract or user.)
        This fallback function is executed whenever the contract receives plain icx coins without data.
        If the fallback function is not decorated with `@payable`,
        it is not listed on the SCORE APIs also cannot be called.
        """
        pass

    @classmethod
    def __get_api(cls) -> dict:
        return getattr(cls, CONST_CLASS_API, "")

    def __validate_external_method(self, func_name: str) -> None:
        """Validate the method indicated by func_name is an external method

        :param func_name: name of method
        """

        if not self.__is_external_method(func_name):
            raise MethodNotFoundException(
                f"Method not found: {type(self).__name__}.{func_name}")

    @classmethod
    def __get_score_element_metadata(cls) -> ScoreElementMetadataContainer:
        return getattr(cls, CONST_CLASS_ELEMENT_METADATA)

    def __call(self,
               func_name: str,
               arg_params: Optional[list] = None,
               kw_params: Optional[dict] = None) -> Any:

        if func_name != STR_FALLBACK and \
                func_name != STR_ON_INSTALL and \
                func_name != STR_ON_UPDATE:
            self.__validate_external_method(func_name)

        if func_name == STR_FALLBACK:
            if not self.__is_payable_method(func_name):
                raise MethodNotFoundException(
                    f"Method not found: {type(self).__name__}.{func_name}")
            score_func = getattr(self, func_name)
            ret = score_func()
        else:
            self.__check_payable(func_name)
            score_func = getattr(self, func_name)
            if arg_params is None:
                arg_params = []
            if kw_params is None:
                kw_params = {}
            ret = score_func(*arg_params, **kw_params)
        return ret

    def __check_payable(self, func_name: str):
        if self.msg.value > 0 and not self.__is_payable_method(func_name):
            raise MethodNotPayableException(
                f"Method not payable: {type(self).__name__}.{func_name}")

    def __is_external_method(self, func_name) -> bool:
        elements = self.__get_score_element_metadata()
        func: FunctionMetadata = elements.get(func_name)
        return isinstance(func, FunctionMetadata) and func.is_external

    def __is_payable_method(self, func_name) -> bool:
        elements = self.__get_score_element_metadata()
        func: FunctionMetadata = elements.get(func_name)
        return isinstance(func, FunctionMetadata) and func.is_payable

    def __is_func_readonly(self, func_name: str) -> bool:
        elements = self.__get_score_element_metadata()
        func: FunctionMetadata = elements.get(func_name)
        return isinstance(func, FunctionMetadata) and func.is_readonly

    def __getattr__(self, item):
        status, ret = AttributeHandler.run(self._context, item)
        if status:
            return ret
        super().__getattribute__(item)

    @property
    def msg(self) -> 'Message':
        """
        Holds information of calling the SCORE

        -  msg.sender : Address of the account who called this function. If
           other contact called this function, msg.sender points to the caller
           contract’s address.

        -  msg.value : Amount of icx that the sender attempts to transfer to the
           current SCORE.
        """
        return self._context.msg

    @property
    def address(self) -> 'Address':
        """
        The current SCORE address

        :return: :class:`.Address` current address
        """
        return self.__address

    @property
    def tx(self) -> 'Transaction':
        """
        Holds information of the transaction

        :return: :class:`.Transaction` transaction
        """
        return self._context.tx

    @property
    def block(self) -> 'Block':
        """
        Deprecated property

        Use block_height and now() instead.
        """
        warnings.warn("Use block_height and now() instead", DeprecationWarning, stacklevel=2)
        return Block(self._context.block.height, self._context.block.timestamp)

    @property
    def db(self) -> 'IconScoreDatabase':
        """
        An instance used to access state DB

        :return: :class:`.IconScoreDatabase` db
        """
        return self.__db

    @property
    def owner(self) -> 'Address':
        """
        Address of the account who deployed the contract

        :return: :class:`.Address` owner address
        """
        return self.__owner

    @property
    def icx(self) -> 'Icx':
        """
        An object used to transfer icx coin

        -  icx.transfer(addr_to(address), amount(integer)) -> bool Transfers
           designated amount of icx coin to ``addr_to``. If exception occurs
           during execution, the exception will be escalated. Returns True if
           coin transfer succeeds.

        -  icx.send(addr_to(address), amount(integer)) -> bool Sends designated
           amount of icx coin to ``addr_to``. Basic behavior is same as
           transfer, the difference is that exception is caught inside the
           function. Returns True when coin transfer succeeded, False when
           failed.

        :return: :class:`.Icx` instance of icx
        """
        if self.__icx is None:
            self.__icx = Icx(self._context, self.__address)
        else:
            # Should update a new context in icx for every tx
            self.__icx._context = self._context

        return self.__icx

    @property
    def block_height(self) -> int:
        """
        Current block height

        :return: current block height
        """
        return self._context.block.height

    def now(self) -> int:
        """
        Timestamp of current block in microseconds

        :return: timestamp in microseconds
        """
        return self._context.block.timestamp

    def call(self, addr_to: 'Address', func_name: str, kw_dict: dict, amount: int = 0):
        """
        Calls an external function provided by another SCORE.
        `func_name` can be `None` if fallback calls

        :param addr_to: :class:`.Address` the address of another SCORE
        :param func_name: function name of another SCORE
        :param kw_dict: arguments of the external function
        :param amount: amount of ICX to transfer in loop
        :return: returning value of the external function
        """
        return InternalCall.message_call(self._context, self.address, addr_to, amount, func_name, None, kw_dict)

    @staticmethod
    def revert(message: Optional[str] = None, code: int = 0):
        """
        Deprecated method

        Use global function `revert()` instead.
        """
        warnings.warn("Use global function revert() instead.", DeprecationWarning, stacklevel=2)
        revert(message, code)

    def get_owner(self, score_address: Optional['Address']) -> Optional['Address']:
        if self._context:
            return self._context.owner
        else:
            return score_address

    @staticmethod
    def create_interface_score(addr_to: 'Address',
                               interface_cls: Callable[['Address'], T]) -> T:
        """
        Creates an object, through which you have an access to the designated SCORE’s external functions.

        :param addr_to: SCORE address
        :param interface_cls: interface class
        :return: An instance of given class
        """
        return create_interface_score(addr_to, interface_cls)

    def get_fee_sharing_proportion(self):
        return self._context.fee_sharing_proportion

    def set_fee_sharing_proportion(self, proportion: int):
        if self._context.type == IconScoreContextType.QUERY:
            raise InvalidRequestException("Cannot set fee sharing proportion in read-only context")
        if proportion < 0 or proportion > 100:
            raise InvalidRequestException("Invalid proportion: should be between 0 and 100")

        self._context.fee_sharing_proportion = proportion

    def deploy(self, tx_hash: bytes):
        warnings.warn("Forbidden function", DeprecationWarning, stacklevel=2)
        if Revision.to_value(self._context.revision) <= Revision.THREE:
            return ChainScore.acceptScore(self._context, self.address, tx_hash)
        else:
            raise AccessDeniedException('No permission')

    def get_tx_hashes_by_score_address(self,
                                       score_address: 'Address') -> Tuple[Optional[bytes], Optional[bytes]]:
        warnings.warn("Forbidden function", DeprecationWarning, stacklevel=2)
        if Revision.to_value(self._context.revision) <= Revision.THREE:
            return ChainScore.addressToTxHashes(self._context, self.address, score_address)
        else:
            raise AccessDeniedException('No permission')

    def get_score_address_by_tx_hash(self,
                                     tx_hash: bytes) -> Optional['Address']:
        warnings.warn("Forbidden function", DeprecationWarning, stacklevel=2)
        if Revision.to_value(self._context.revision) <= Revision.THREE:
            return ChainScore.txHashToAddress(self._context, self.address, tx_hash)
        else:
            raise AccessDeniedException('No permission')


class AttributeHandler(object):
    MECA_COIN_CODEHASH = '0x5edae375f569d1243b1d9241344d3c2cd34e0dc232242060b0a7508a8335b8dc'

    @classmethod
    def run(cls, context, item):
        if item == 'privateSaleHolder' and \
                os.path.basename(context.code) == cls.MECA_COIN_CODEHASH:
            return True, ""

        return False, None
