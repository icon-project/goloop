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

from typing import List
from typing_extensions import TypedDict

from .icon_score_base import interface
from .icon_score_base2 import InterfaceScore
from ..base.address import Address


class Delegation(TypedDict):
    address: Address
    value: int


class InterfaceSystemScore(InterfaceScore):
    @interface
    def setStake(self, value: int) -> None: pass

    @interface
    def getStake(self, address: Address) -> dict: pass

    @interface
    def estimateUnstakeLockPeriod(self) -> dict: pass

    @interface
    def setDelegation(self, delegations: List[Delegation] = None): pass

    @interface
    def getDelegation(self, address: Address) -> dict: pass

    @interface
    def claimIScore(self): pass

    @interface
    def queryIScore(self, address: Address) -> dict: pass

    @interface
    def getIISSInfo(self) -> dict: pass

    @interface
    def getPRep(self, address: Address) -> dict: pass

    @interface
    def getPReps(self, startRanking: int, endRanking: int) -> list: pass

    @interface
    def getMainPReps(self) -> dict: pass

    @interface
    def getSubPReps(self) -> dict: pass

    @interface
    def getPRepTerm(self) -> dict: pass

    @interface
    def getScoreDepositInfo(self, address: Address) -> dict: pass

    @interface
    def burn(self): pass

    @interface
    def getScoreOwner(score: Address) -> Address: pass

    @interface
    def setScoreOwner(score: Address, owner: Address): pass
