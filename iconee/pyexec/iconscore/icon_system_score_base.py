# Copyright 2021 ICON Foundation
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

from abc import abstractmethod
from typing import Optional, Tuple, Dict, Any, List

from .internal_call import ChainScore
from ..base.address import Address, BUILTIN_SCORE_ADDRESS_MAPPER
from ..base.exception import AccessDeniedException, IconServiceBaseException, InvalidRequestException, \
    InvalidParamsException
from ..database.db import IconScoreDatabase
from ..icon_constant import IconServiceFlag, Revision, IconNetworkValueType
from ..iconscore.icon_score_base import IconScoreBase


class IconScoreDeployTXParams(object):

    def __init__(self, tx_hash: bytes, score_address: 'Address'):
        self._tx_hash = tx_hash
        self._score_address = score_address

    @property
    def tx_hash(self) -> bytes:
        return self._tx_hash

    @property
    def score_address(self) -> 'Address':
        return self._score_address


class IconScoreDeployInfo(object):

    def __init__(self, score_address: 'Address', current_tx_hash: bytes, next_tx_hash: bytes):
        self._score_address = score_address
        self._current_tx_hash = current_tx_hash
        self._next_tx_hash = next_tx_hash

    @property
    def score_address(self) -> 'Address':
        return self._score_address

    @property
    def current_tx_hash(self) -> bytes:
        return self._current_tx_hash

    @property
    def next_tx_hash(self) -> bytes:
        return self._next_tx_hash


class IconSystemScoreBase(IconScoreBase):

    @abstractmethod
    def on_install(self, **kwargs) -> None:
        super().on_install(**kwargs)

    @abstractmethod
    def on_update(self, **kwargs) -> None:
        super().on_update(**kwargs)

    @abstractmethod
    def __init__(self, db: 'IconScoreDatabase') -> None:
        super().__init__(db)
        if not self.is_builtin_score(self.address):
            raise AccessDeniedException(f"Not a builtin SCORE")

    @staticmethod
    def is_builtin_score(score_address: 'Address') -> bool:
        return score_address in BUILTIN_SCORE_ADDRESS_MAPPER.values()

    def get_icon_service_flag(self) -> int:
        return IconServiceFlag.FEE \
               | IconServiceFlag.AUDIT \
               | IconServiceFlag.SCORE_PACKAGE_VALIDATOR

    def get_deploy_tx_params(self, tx_hash: bytes) -> Optional['IconScoreDeployTXParams']:
        address = ChainScore.txHashToAddress(self._context, self.address, tx_hash)
        if address is not None:
            return IconScoreDeployTXParams(tx_hash, address)
        return None

    def get_deploy_info(self, score_address: 'Address') -> Optional['IconScoreDeployInfo']:
        current, _next = ChainScore.addressToTxHashes(self._context, self.address, score_address)
        return IconScoreDeployInfo(score_address, current, _next)

    def is_score_active(self, score_address: 'Address') -> bool:
        status = ChainScore.getScoreStatus(self._context, self.address, score_address)
        if status is None:
            return False
        current = status.get('current', None)
        return current is not None and current['status'] == 'active'

    def get_score_status(self, score_address: 'Address') -> Optional[dict]:
        return ChainScore.getScoreStatus(self._context, self.address, score_address)

    def disqualify_prep(self, address: 'Address') -> Tuple[bool, str]:
        success: bool = True
        reason: str = ""
        try:
            ChainScore.disqualify_prep(self._context, self.address, address)
        except IconServiceBaseException as e:
            success = False
            reason = str(e)
        finally:
            return success, reason

    def validate_irep(self, irep: int):
        if Revision.to_value(self._context.revision) < Revision.SET_IREP_VIA_NETWORK_PROPOSAL:
            raise InvalidRequestException(f"Can't register I-Rep proposal. Revision must be larger than "
                                          f"{Revision.SET_IREP_VIA_NETWORK_PROPOSAL.value - 1}")
        ChainScore.validate_irep(self._context, self.address, irep)

    def migrate_icon_network_value(self, data: Dict['IconNetworkValueType', Any]):
        for type_, value in data.items():
            if type_ == IconNetworkValueType.SCORE_BLACK_LIST:
                items: List['Address'] = value
                for item in items:
                    ChainScore.blockScore(self._context, self.address, item)

    def get_icon_network_value(self, type_: 'IconNetworkValueType') -> Any:
        if type_ == IconNetworkValueType.STEP_PRICE:
            return ChainScore.getStepPrice(self._context, self.address)
        elif type_ == IconNetworkValueType.STEP_COSTS:
            return ChainScore.getStepCosts(self._context, self.address)
        elif type_ == IconNetworkValueType.MAX_STEP_LIMITS:
            return {
                'invoke': ChainScore.getMaxStepLimit(self._context, self.address, 'invoke'),
                'query': ChainScore.getMaxStepLimit(self._context, self.address, 'query')
            }
        elif type_ == IconNetworkValueType.SCORE_BLACK_LIST:
            return ChainScore.getBlockedScores(self._context, self.address)
        elif type_ == IconNetworkValueType.IMPORT_WHITE_LIST:
            return {'iconservice': ['*']}
        elif type_ == IconNetworkValueType.SERVICE_CONFIG:
            return self.get_icon_service_flag()
        elif type_ == IconNetworkValueType.REVISION_CODE:
            return ChainScore.getRevision(self._context, self.address)
        elif type_ == IconNetworkValueType.REVISION_NAME:
            rev = ChainScore.getRevision(self._context, self.address)
            return f'Revision{rev}'
        elif type_ == IconNetworkValueType.IREP:
            return ChainScore.getIRep(self._context, self.address)
        else:
            raise InvalidParamsException(f"Invalid INV type: {type_}")

    def set_icon_network_value(self, type_: 'IconNetworkValueType', value: Any):
        if type_ not in IconNetworkValueType:
            raise InvalidParamsException(f"Invalid INV type: {type_}")

        if type_ == IconNetworkValueType.STEP_PRICE:
            return ChainScore.setStepPrice(self._context, self.address, value)
        elif type_ == IconNetworkValueType.SCORE_BLACK_LIST:
            stored = ChainScore.getBlockedScores(self._context, self.address)
            if len(stored) < len(value):
                added = set(value).difference(set(stored))
                ChainScore.blockScore(self._context, self.address, list(added)[0])
            elif len(stored) > len(value):
                removed = set(stored).difference(set(value))
                ChainScore.unblockScore(self._context, self.address, list(removed)[0])
        elif type_ == IconNetworkValueType.REVISION_CODE:
            return ChainScore.setRevision(self._context, self.address, value)
        elif type_ == IconNetworkValueType.IREP:
            return ChainScore.setIRep(self._context, self.address, value)
        elif type_ == IconNetworkValueType.STEP_COSTS:
            for k, v in value.items():
                ChainScore.setStepCost(self._context, self.address, k, v)

    def apply_revision_change(self, revision: int):
        # just for backward compatibility, no action needed
        pass

    def validate_reward_fund(self, iglobal: int):
        if Revision.to_value(self._context.revision) < Revision.ICON2:
            raise InvalidRequestException(f"Can't register Monthly Reward Fund Setting Proposal. "
                                          f"Revision must be larger than {Revision.ICON2.value - 1}")
        ChainScore.validate_reward_fund(self._context, self.address, iglobal)

    def set_reward_fund(self, iglobal: int):
        if Revision.to_value(self._context.revision) < Revision.ICON2:
            raise InvalidRequestException(f"Can't register Monthly Reward Fund Setting Proposal. "
                                          f"Revision must be larger than {Revision.ICON2.value - 1}")
        ChainScore.setRewardFund(self._context, self.address, iglobal)

    def set_reward_fund_allocation(self, iprep: int, icps: int, irelay: int, ivoter: int):
        if Revision.to_value(self._context.revision) < Revision.ICON2:
            raise InvalidRequestException(f"Can't register Monthly Reward Fund Setting Proposal. "
                                          f"Revision must be larger than {Revision.ICON2.value - 1}")
        ChainScore.setRewardFundAllocation(self._context, self.address, iprep, icps, irelay, ivoter)
