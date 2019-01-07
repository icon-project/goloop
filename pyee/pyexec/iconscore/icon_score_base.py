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

import warnings
from abc import abstractmethod, ABC, ABCMeta
from functools import partial, wraps
from inspect import isfunction, getmembers, signature, Parameter
from typing import TYPE_CHECKING, Callable, Any, List, Tuple, Optional, Union

from ..base.exception import *
from ..database.db import IconScoreDatabase, DatabaseObserver
from ..base.transaction import Transaction
from ..base.message import Message
from ..base.address import Address
from ..icon_constant import ICX_TRANSFER_EVENT_LOG, IconScoreContextType
from ..utils import get_main_type_from_annotations_type

from .icon_score_base2 import InterfaceScore, revert, Block, Icx
from .icon_score_constant import CONST_INDEXED_ARGS_COUNT, CONST_BIT_FLAG, ConstBitFlag, \
    FORMAT_IS_NOT_FUNCTION_OBJECT, FORMAT_IS_NOT_DERIVED_OF_OBJECT, FORMAT_DECORATOR_DUPLICATED, \
    STR_FALLBACK, STR_ON_INSTALL, STR_ON_UPDATE, \
    CONST_CLASS_EXTERNALS, CONST_CLASS_PAYABLES, CONST_CLASS_API, T, BaseType
from .icon_score_context import ContextGetter, IconScoreContext
from .score_api_generator import ScoreApiGenerator
from .icon_score_step import StepType
from .icon_score_eventlog import EventLogEmitter
from .internal_call import InternalCall

INDEXED_ARGS_LIMIT = 3


def interface(func):
    """interface decorator

    :param func:
    :return:
    """
    cls_name, func_name = str(func.__qualname__).split('.')
    if not isfunction(func):
        raise InterfaceException(FORMAT_IS_NOT_FUNCTION_OBJECT.format(func, cls_name))

    if getattr(func, CONST_BIT_FLAG, 0) & ConstBitFlag.Interface:
        raise IconScoreException(FORMAT_DECORATOR_DUPLICATED.format('interface', func_name, cls_name))

    bit_flag = getattr(func, CONST_BIT_FLAG, 0) | ConstBitFlag.Interface
    setattr(func, CONST_BIT_FLAG, bit_flag)

    @wraps(func)
    def __wrapper(calling_obj: Any, *args, **kwargs):
        if not isinstance(calling_obj, InterfaceScore):
            raise InterfaceException(FORMAT_IS_NOT_DERIVED_OF_OBJECT.format(InterfaceScore.__name__))

        score = calling_obj.from_score
        addr_to = calling_obj.addr_to

        if addr_to is None:
            raise InterfaceException('Can\'t create an interface SCORE with None address')

        return InternalCall.message_call(score._context, score.address, addr_to, 0, func_name, args, kwargs)

    return __wrapper


def eventlog(func=None, *, indexed=0):
    """eventlog decorator

    :param func: function name
    :param indexed: index args
    """
    if func is None:
        return partial(eventlog, indexed=indexed)

    cls_name, func_name = str(func.__qualname__).split('.')
    if not isfunction(func):
        raise EventLogException(
            FORMAT_IS_NOT_FUNCTION_OBJECT.format(func, cls_name))

    if not list(signature(func).parameters.keys())[0] == 'self':
        raise EventLogException("define 'self' as the first parameter in the event log")
    if indexed > INDEXED_ARGS_LIMIT:
        raise EventLogException(
            f'indexed arguments are overflow: limit={INDEXED_ARGS_LIMIT}')

    parameters = signature(func).parameters.values()
    if len(parameters) - 1 < indexed:
        raise EventLogException("index exceeds the number of parameters")

    if getattr(func, CONST_BIT_FLAG, 0) & ConstBitFlag.EventLog:
        raise IconScoreException(
            FORMAT_DECORATOR_DUPLICATED.format('eventlog', func_name, cls_name))

    bit_flag = getattr(func, CONST_BIT_FLAG, 0) | ConstBitFlag.EventLog
    setattr(func, CONST_BIT_FLAG, bit_flag)
    setattr(func, CONST_INDEXED_ARGS_COUNT, indexed)

    event_signature = __retrieve_event_signature(func_name, parameters)

    @wraps(func)
    def __wrapper(calling_obj: Any, *args, **kwargs):
        if not (isinstance(calling_obj, IconScoreBase)):
            raise EventLogException(
                FORMAT_IS_NOT_DERIVED_OF_OBJECT.format(IconScoreBase.__name__))
        try:
            arguments = __resolve_arguments(func_name, parameters, args, kwargs)
        except IconTypeError as e:
            raise EventLogException(e.message)

        if event_signature == ICX_TRANSFER_EVENT_LOG:
            # 'ICXTransfer(Address,Address,int)' is reserved
            raise EventLogException(
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
                raise IconTypeError(
                    f"Missing argument hint for '{function_name}': '{param.name}'")

            main_type = None
            if isinstance(param.annotation, type):
                main_type = param.annotation
            elif param.annotation == 'Address':
                main_type = Address

            # Raises an exception if the types are not supported
            if main_type is None or not issubclass(main_type, BaseType.__constraints__):
                raise IconTypeError(
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
        raise EventLogException(
            f"exceed the maximum number of arguments which event log method({function_name}) can accept")

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
                raise IconTypeError(
                    f"Duplicated argument value for '{function_name}': {name}")
        else:
            # If arg is over, the argument should be searched on kwargs
            try:
                value = kwargs[name]
            except KeyError:
                if not parameter.default == Parameter.empty:
                    value = parameter.default
                else:
                    raise IconTypeError(
                        f"Missing argument value for '{function_name}': {name}")

        main_type = get_main_type_from_annotations_type(annotation)

        if main_type == 'Address':
            main_type = Address

        if value is not None and not isinstance(value, main_type):
            raise IconTypeError(f"Mismatch type type of '{name}': "
                                f"{type(value)}, expected: {main_type}")
        arguments.append(value)
    return arguments


def external(func=None, *, readonly=False):
    if func is None:
        return partial(external, readonly=readonly)

    cls_name, func_name = str(func.__qualname__).split('.')
    if not isfunction(func):
        raise IconScoreException(FORMAT_IS_NOT_FUNCTION_OBJECT.format(func, cls_name))

    if func_name == STR_FALLBACK:
        raise IconScoreException(f"can't locate external to this func func: {func_name}, cls: {cls_name}")

    if getattr(func, CONST_BIT_FLAG, 0) & ConstBitFlag.External:
        raise IconScoreException(FORMAT_DECORATOR_DUPLICATED.format('external', func_name, cls_name))

    bit_flag = getattr(func, CONST_BIT_FLAG, 0) | ConstBitFlag.External | int(readonly)
    setattr(func, CONST_BIT_FLAG, bit_flag)

    @wraps(func)
    def __wrapper(calling_obj: Any, *args, **kwargs):
        if not (isinstance(calling_obj, IconScoreBase)):
            raise ExternalException(
                FORMAT_IS_NOT_DERIVED_OF_OBJECT.format(IconScoreBase.__name__), func_name, cls_name)
        res = func(calling_obj, *args, **kwargs)
        return res

    return __wrapper


def payable(func):
    cls_name, func_name = str(func.__qualname__).split('.')
    if not isfunction(func):
        raise IconScoreException(FORMAT_IS_NOT_FUNCTION_OBJECT.format(func, cls_name))

    if getattr(func, CONST_BIT_FLAG, 0) & ConstBitFlag.Payable:
        raise IconScoreException(FORMAT_DECORATOR_DUPLICATED.format('payable', func_name, cls_name))

    bit_flag = getattr(func, CONST_BIT_FLAG, 0) | ConstBitFlag.Payable
    setattr(func, CONST_BIT_FLAG, bit_flag)

    @wraps(func)
    def __wrapper(calling_obj: Any, *args, **kwargs):

        if not (isinstance(calling_obj, IconScoreBase)):
            raise PayableException(
                FORMAT_IS_NOT_DERIVED_OF_OBJECT.format(IconScoreBase.__name__), func_name, cls_name)
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
        if IconScoreObject in bases:
            return super().__new__(mcs, name, bases, namespace, **kwargs)

        cls = super().__new__(mcs, name, bases, namespace, **kwargs)

        if not isinstance(namespace, dict):
            raise IconScoreException('attr is not dict!')

        custom_funcs = [value for key, value in getmembers(cls, predicate=isfunction)
                        if not key.startswith('__')]

        external_funcs = {func.__name__: signature(func) for func in custom_funcs
                          if getattr(func, CONST_BIT_FLAG, 0) & ConstBitFlag.External}
        payable_funcs = [func for func in custom_funcs
                         if getattr(func, CONST_BIT_FLAG, 0) & ConstBitFlag.Payable]

        readonly_payables = [func for func in payable_funcs
                             if getattr(func, CONST_BIT_FLAG, 0) & ConstBitFlag.ReadOnly]

        if bool(readonly_payables):
            raise IconScoreException(f"Readonly method cannot be payable: {readonly_payables}")

        if external_funcs:
            setattr(cls, CONST_CLASS_EXTERNALS, external_funcs)
        if payable_funcs:
            payable_funcs = {func.__name__: signature(func) for func in payable_funcs}
            setattr(cls, CONST_CLASS_PAYABLES, payable_funcs)

        # Generate SCORE API list
        api_list = ScoreApiGenerator.generate(custom_funcs)
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
        Invoked when the contract is deployed for the first time, and will not be called again on contract update or deletion afterward.
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
        """
        super().__init__(db)
        self.__db = db
        self.__address = db.address
        self.__owner = self.get_owner(self.__address)
        self.__icx = None

        if not self.__get_attr_dict(CONST_CLASS_EXTERNALS):
            raise ExternalException('this score has no external functions', '__init__', str(type(self)))

        self.__db.set_observer(self.__create_db_observer())

    def fallback(self) -> None:
        """
        fallback function can not be decorated with `@external`. (i.e., fallback function is not allowed to be called by external contract or user.)
        This fallback function is executed whenever the contract receives plain icx coins without data.
        If the fallback function is not decorated with `@payable`, the icx coin transfers to the contract will fail.
        """
        pass

    @classmethod
    def get_api(cls) -> dict:
        return getattr(cls, CONST_CLASS_API, "")

    def validate_external_method(self, func_name: str) -> None:
        """Validate the method indicated by func_name is an external method

        :param func_name: name of method
        """

        if func_name not in self.__get_attr_dict(CONST_CLASS_EXTERNALS):
            raise ExternalException(f"Invalid external method",
                                    func_name,
                                    type(self).__name__,
                                    ExceptionCode.METHOD_NOT_FOUND)

    @classmethod
    def __get_attr_dict(cls, attr: str) -> dict:
        return getattr(cls, attr, {})

    def __create_db_observer(self) -> 'DatabaseObserver':
        return DatabaseObserver(
            self.__on_db_get, self.__on_db_put, self.__on_db_delete)

    def __call(self,
               func_name: str,
               arg_params: Optional[list] = None,
               kw_params: Optional[dict] = None) -> Any:

        if func_name != STR_FALLBACK and \
                func_name != STR_ON_INSTALL and \
                func_name != STR_ON_UPDATE:
            self.validate_external_method(func_name)

        self.__check_payable(func_name, self.__get_attr_dict(CONST_CLASS_PAYABLES))

        score_func = getattr(self, func_name)

        if func_name == STR_FALLBACK:
            ret = score_func()
        else:
            if arg_params is None:
                arg_params = []
            if kw_params is None:
                kw_params = {}
            ret = score_func(*arg_params, **kw_params)

        return ret

    def __check_payable(self, func_name: str, payable_dict: dict):
        if func_name not in payable_dict:
            if self.msg.value > 0:
                raise PayableException(f"This is not payable", func_name, type(self).__name__)

    def __is_func_readonly(self, func_name: str) -> bool:
        func = getattr(self, func_name)
        return bool(getattr(func, CONST_BIT_FLAG, 0) & ConstBitFlag.ReadOnly)

    @staticmethod
    def __on_db_get(context: 'IconScoreContext',
                    key: bytes,
                    value: bytes):
        """Invoked when `get` is called in `ContextDatabase`.

        # All steps are managed in the score
        # Don't move to another codes

        :param context: SCORE context
        :param key: key
        :param value: value
        """

        if context and context.step_counter and \
                context.type != IconScoreContextType.DIRECT:
            length = 1
            if value:
                length = len(value)
            context.step_counter.apply_step(StepType.GET, length)

    @staticmethod
    def __on_db_put(context: 'IconScoreContext',
                    key: bytes,
                    old_value: bytes,
                    new_value: bytes):
        """Invoked when `put` is called in `ContextDatabase`.

        # All steps are managed in the score
        # Don't move to another codes

        :param context: SCORE context
        :param key: key
        :param old_value: old value
        :param new_value: new value
        """

        if context and context.step_counter and not context.readonly:
            if old_value:
                # modifying a value
                context.step_counter.apply_step(
                    StepType.REPLACE, len(new_value))
            else:
                # newly storing a value
                context.step_counter.apply_step(
                    StepType.SET, len(new_value))

    @staticmethod
    def __on_db_delete(context: 'IconScoreContext',
                       key: bytes,
                       old_value: bytes):
        """Invoked when `delete` is called in `ContextDatabase`.

        # All steps are managed in the score
        # Don't move to another codes
        :param context: SCORE context
        :param key: key
        :param old_value: old value
        """

        if context and context.step_counter and not context.readonly:
            context.step_counter.apply_step(
                StepType.DELETE, len(old_value))

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
        Call external function provided by other IconScore with arguments without fallback

        :param addr_to: :class:`.Address` the address of other IconScore
        :param func_name: function name provided by other IconScore
        :param kw_dict: Arguments of the external function
        :param amount: ICX value to enclose with. in loop.
        :return: returning value of the external function
        """
        warnings.warn('Use create_interface_score() instead.', DeprecationWarning, stacklevel=2)
        return InternalCall.message_call(self._context, self.address, addr_to, amount, func_name, None, kw_dict)

    @staticmethod
    def revert(message: Optional[str] = None,
               code: Union[ExceptionCode, int] = ExceptionCode.SCORE_ERROR) -> None:
        revert(message, code)

    def get_owner(self, score_address: Optional['Address']) -> Optional['Address']:
        if self._context:
            return self._context.owner
        else:
            return score_address

    def create_interface_score(self,
                               addr_to: 'Address',
                               interface_cls: Callable[['Address', callable], T]) -> T:
        """
        Creates an object, through which you have an access to the designated SCORE’s external functions.

        :param addr_to: SCORE address
        :param interface_cls: interface class
        :return: An instance of given class
        """

        if interface_cls is InterfaceScore:
            raise InterfaceException(FORMAT_IS_NOT_DERIVED_OF_OBJECT.format(InterfaceScore.__name__))
        return interface_cls(addr_to, self)
