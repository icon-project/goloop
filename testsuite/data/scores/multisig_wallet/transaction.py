# -*- coding: utf-8 -*-

# Copyright 2018 ICON Foundation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from iconservice import *

# address, value fix
ADDRESS_BYTE_LEN = 21
DEFAULT_VALUE_BYTES = 16
DATA_BYTE_ORDER = "big"

MAX_METHOD_LEN = 100
MAX_PARAMS_LEN = 1000
MAX_DESCRIPTION_LEN = 1000


class Transaction:
    def __init__(self,
                 destination: Address,
                 method: str,
                 params: str,
                 value: int,
                 description: str,
                 executed: bool):

        self._executed = executed
        self._destination = destination
        self._value = value
        self._method = method
        self._params = params
        self._description = description

    @property
    def executed(self) -> bool:
        return self._executed

    @executed.setter
    def executed(self, executed: bool):
        self._executed = executed

    @property
    def destination(self) -> Address:
        return self._destination

    @property
    def method(self) -> str:
        return self._method

    @property
    def params(self) -> str:
        return self._params

    @property
    def value(self) -> int:
        return self._value

    @property
    def description(self) -> str:
        return self._description

    def to_dict(self):
        tx_dict = self.__dict__
        tx_dict['_destination'] = str(self.destination)
        return tx_dict

    @classmethod
    def create_transaction_with_validation(cls,
                                           destination: Address,
                                           method: str,
                                           params: str,
                                           value: int,
                                           description: str,
                                           executed: bool = False):
        # as None type can't be converted to bytes, must be changed to ""
        method = "" if method is None else method
        params = "" if params is None else params

        if len(method) > MAX_METHOD_LEN \
                or len(params) > MAX_PARAMS_LEN \
                or len(description) > MAX_DESCRIPTION_LEN:
            revert("too long parameter length")
        try:
            value.to_bytes(DEFAULT_VALUE_BYTES, DATA_BYTE_ORDER)
        except OverflowError:
            revert("exceed ICX amount you can send at one time")

        return cls(executed=executed,
                   destination=destination,
                   value=value,
                   method=method,
                   params=params,
                   description=description)

    @classmethod
    def from_bytes(cls, buf: bytes):
        encoded_executed = bool(buf[0])
        encoded_destination = buf[1: 1 + ADDRESS_BYTE_LEN]
        encoded_value = buf[1 + ADDRESS_BYTE_LEN: 1 + ADDRESS_BYTE_LEN + DEFAULT_VALUE_BYTES]
        flexible_vars_json_string = buf[1 + ADDRESS_BYTE_LEN + DEFAULT_VALUE_BYTES:].decode()
        flexible_vars_json = json_loads(flexible_vars_json_string)

        return cls(executed=encoded_executed,
                   destination=Address.from_bytes(encoded_destination),
                   value=int.from_bytes(encoded_value, DATA_BYTE_ORDER),
                   method=flexible_vars_json["method"],
                   params=flexible_vars_json["params"],
                   description=flexible_vars_json["description"])

    def to_bytes(self) -> bytes:
        encoded_executed = self.executed.to_bytes(1, DATA_BYTE_ORDER)
        encoded_value = self.value.to_bytes(DEFAULT_VALUE_BYTES, DATA_BYTE_ORDER)
        destination_bytes = self.destination.to_bytes()
        destination_bytes = destination_bytes if len(destination_bytes) == ADDRESS_BYTE_LEN \
            else b'\x00' + destination_bytes

        flexible_vars = dict()
        flexible_vars["method"] = self.method
        flexible_vars["params"] = self.params
        flexible_vars["description"] = self.description

        encoded_flexible_vars_json = json_dumps(flexible_vars).encode(encoding="utf-8")
        return encoded_executed + destination_bytes + encoded_value + encoded_flexible_vars_json
