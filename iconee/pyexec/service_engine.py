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

from .base.address import Address, GETAPI_DUMMY_ADDRESS
from .base.exception import *
from .base.type_converter import TypeConverter
from .database.factory import ContextDatabaseFactory
from .icon_constant import Status
from .iconscore.icon_score_base import IconScoreBase
from .iconscore.icon_score_context import ContextContainer, IconScoreContext
from .iconscore.icon_score_eventlog import EventLogEmitter
from .iconscore.icon_score_mapper import IconScoreMapper
from .iconscore.internal_call import InternalCall
from .iconscore.score_package_validator import ScorePackageValidator
from .logger import Logger, SystemLogger

TAG = 'ServiceEngine'


def decode_params(values: dict) -> dict:
    result = {}
    if isinstance(values, dict):
        for k, v in values.items():
            new_key = k
            if isinstance(k, bytes):
                new_key = k.decode()
            elif not isinstance(k, str):
                raise BaseException('Unexpected key type')

            if isinstance(v, bytes):
                result[new_key] = v.decode()
            else:
                result[new_key] = v
    return result


class ServiceEngine(ContextContainer):

    _score_mapper = None
    _proxy = None

    @classmethod
    def open(cls, proxy):
        cls._score_mapper = IconScoreMapper()
        cls._proxy = proxy
        ContextDatabaseFactory.open(proxy, ContextDatabaseFactory.Mode.SINGLE_DB)
        EventLogEmitter.open(proxy)
        InternalCall.open(proxy)

    @classmethod
    def invoke(cls, context: IconScoreContext):
        Logger.debug(f'[invoke] {context.method}, {context.params}', TAG)

        cls._push_context(context)
        status, step_used, ret = cls._handle_invoke(context)
        cls._pop_context()

        Logger.debug(f'*** RESULT: {status}, {step_used}, {ret}', TAG)
        return status, step_used, ret

    @classmethod
    def get_score_api(cls, code: str, verify_package=False):
        try:
            if verify_package:
                ScorePackageValidator.execute(code)
            icon_score: 'IconScoreBase' = cls._get_icon_score(GETAPI_DUMMY_ADDRESS, code)
            get_api = getattr(icon_score, '_IconScoreBase__get_api')
            ret = get_api()
            status = Status.SUCCESS
        except BaseException as e:
            status, ret = cls._get_status_from_exception(e)

        return status, ret

    @classmethod
    def _get_icon_score(cls, address: Address, code: str):
        return cls._score_mapper.get_icon_score(address, code)

    @classmethod
    def _handle_invoke(cls, context):
        try:
            ret = cls._internal_call(context)
            status = Status.SUCCESS
        except BaseException as e:
            status, ret = cls._get_status_from_exception(e)
        finally:
            cls._proxy.handle_set_values()
            if context.fee_sharing_proportion > 0:
                cls._proxy.set_fee_proportion(context.fee_sharing_proportion)
            step_used = context.step_counter.step_used

        return status, step_used, ret

    @classmethod
    def _internal_call(cls, context: IconScoreContext):
        icon_score: 'IconScoreBase' = cls._get_icon_score(context.to, context.code)
        if icon_score is None:
            raise ScoreNotFoundException(f'SCORE not found: {context.to}')

        func_name: str = context.method
        context.set_func_type_by_icon_score(icon_score, func_name)

        if isinstance(context.params, dict):
            arg_params = []
            params: dict = decode_params(context.params)
            kw_params = cls._convert_score_params_by_annotations(icon_score, func_name, params)
            Logger.debug(f'kw_params: {kw_params}', TAG)
        elif isinstance(context.params, list):
            arg_params: list = context.params
            Logger.debug(f'arg_params: {arg_params}', TAG)
            kw_params = {}
        else:
            raise InvalidParamsException('Unknown params type')

        score_func = getattr(icon_score, '_IconScoreBase__call')
        return score_func(func_name=func_name, arg_params=arg_params, kw_params=kw_params)

    @staticmethod
    def _convert_score_params_by_annotations(icon_score: 'IconScoreBase', func_name: str, kw_params: dict) -> dict:
        tmp_params = kw_params
        score_func = getattr(icon_score, func_name)
        annotation_params = TypeConverter.make_annotations_from_method(score_func)
        TypeConverter.convert_data_params(annotation_params, tmp_params)
        return tmp_params

    @classmethod
    def _get_status_from_exception(cls, e: BaseException):
        if isinstance(e, IconServiceBaseException):
            if isinstance(e, IconScoreException):
                tag = 'ScoreException'
            else:
                tag = 'SystemException'
            Logger.exception(e.message, tag)

            code = e.code
            message = e.message
        else:
            SystemLogger.exception(repr(e), 'SystemError')

            code = ExceptionCode.SYSTEM_ERROR
            message = str(e)

        return code, message
