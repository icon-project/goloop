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

"""functions and classes to handle address
"""

import hashlib
from enum import IntEnum
from typing import Optional

from ..icon_constant import DATA_BYTE_ORDER
from ..utils import is_lowercase_hex_string, int_to_bytes
from .exception import InvalidParamsException

ICON_EOA_ADDRESS_PREFIX = 'hx'
ICON_CONTRACT_ADDRESS_PREFIX = 'cx'
ICON_ADDRESS_BODY_SIZE = 20
ICON_ADDRESS_BYTES_SIZE = 21


def is_icon_address_valid(address: str) -> bool:
    """Check whether address is in icon address format or not

    :param address: (str) address string including prefix
    :return: (bool)
    """
    try:
        if isinstance(address, str) and len(address) == 42:
            prefix, body = split_icon_address(address)
            if prefix == ICON_EOA_ADDRESS_PREFIX or \
                    prefix == ICON_CONTRACT_ADDRESS_PREFIX:
                return is_lowercase_hex_string(body)
    finally:
        pass

    return False


def split_icon_address(address: str) -> (str, str):
    """Split icon address into 2-char prefix and 40-char address body

    :param address: 42-char address string
    :return: prefix, body
    """
    return address[:2], address[2:]


class AddressPrefix(IntEnum):
    """
    Enumeration of Address prefix

    - EOA: Externally Owned Account
    - CONTRACT: Contract Account
    """
    EOA = 0
    CONTRACT = 1

    def __str__(self) -> str:
        if self == AddressPrefix.EOA:
            return ICON_EOA_ADDRESS_PREFIX
        if self == AddressPrefix.CONTRACT:
            return ICON_CONTRACT_ADDRESS_PREFIX

    @staticmethod
    def from_string(prefix: str):
        """
        Returns address prefix enumerator

        :param prefix: 2-byte address prefix (hx or cx)
        :return: (AddressPrefix) address prefix enumerator
        """
        if prefix == ICON_EOA_ADDRESS_PREFIX:
            return AddressPrefix.EOA
        if prefix == ICON_CONTRACT_ADDRESS_PREFIX:
            return AddressPrefix.CONTRACT

        raise InvalidParamsException('Invalid address prefix')


class Address(object):
    """Address class
    """

    def __init__(self,
                 address_prefix: AddressPrefix,
                 address_body: bytes, ignore_length_validate: bool = False) -> None:
        """Constructor

        :param address_prefix: address prefix enumerator
        :param address_body: 20-byte address body
        """

        if not isinstance(address_prefix, AddressPrefix):
            raise InvalidParamsException('Invalid address prefix type')
        if not isinstance(address_body, bytes):
            raise InvalidParamsException('Invalid address body type')

        if not ignore_length_validate:
            if len(address_body) != ICON_ADDRESS_BODY_SIZE:
                raise InvalidParamsException('Address length is not 20 in bytes')

        self.__prefix = address_prefix
        self.__body = address_body

    @property
    def prefix(self) -> AddressPrefix:
        """Returns address prefix part

        :return: :class:`.AddressPrefix` AddressPrefix.EOA(0) or AddressPrefix.CONTRACT(1)
        """
        return self.__prefix

    @property
    def body(self) -> bytes:
        """Returns 20-byte address body part

        :return: 20 byte data standing for address
        """
        return self.__body

    def __eq__(self, other) -> bool:
        """operator == overriding

        :return: bool
        """
        return \
            isinstance(other, Address) \
            and self.__prefix == other.prefix \
            and self.__body == other.body

    def __ne__(self, other) -> bool:
        """operator != overriding

        :return: (bool)
        """
        return not self.__eq__(other)

    def __str__(self) -> str:
        """operator str() overriding

        returns prefix(2) + 40-char hexadecimal address

        :return: (str) 42-char address
        """
        return f'{str(self.prefix)}{self.body.hex()}'

    def __repr__(self) -> str:
        return self.__str__()

    def __hash__(self) -> int:
        """Returns a hash value for this object

        :return: hash value
        """
        return hash(self.__prefix.to_bytes(1, DATA_BYTE_ORDER) + self.__body)

    @property
    def is_contract(self) -> bool:
        """
        Whether the address is SCORE

        :return: True(contract) False(Not contract)
        """
        return self.prefix == AddressPrefix.CONTRACT

    @staticmethod
    def from_string(address: str) -> 'Address':
        """
        Creates an Address object from given 42-char string `address`

        :return: :class:`.Address`
        """

        if not is_icon_address_valid(address):
            raise InvalidParamsException('Invalid address')

        prefix, body = split_icon_address(address)

        address_prefix = AddressPrefix.from_string(prefix)
        address_body = bytes.fromhex(body)

        return Address(address_prefix, address_body)

    @staticmethod
    def from_data(prefix: AddressPrefix, data: bytes) -> 'Address':
        """
        Creates an Address object using given bytes data

        :param prefix: address prefix
        :param data: arbitrary bytes data
        :return: :class:`.Address`
        """
        hash_value = hashlib.sha3_256(data).digest()
        return Address(prefix, hash_value[-20:])

    @staticmethod
    def from_bytes(buf: bytes) -> Optional['Address']:
        """
        Creates an Address object from given raw bytes that represent address

        :param buf: :class:`.bytes` raw bytes data
        :return: :class:`.Address`
        """
        if not isinstance(buf, bytes):
            return None

        buf_size = len(buf)
        if buf_size not in (ICON_ADDRESS_BODY_SIZE, ICON_ADDRESS_BYTES_SIZE):
            return None

        prefix = AddressPrefix.EOA
        if buf_size != ICON_ADDRESS_BODY_SIZE:
            prefix_byte = buf[0:1]
            prefix_int = int.from_bytes(prefix_byte, DATA_BYTE_ORDER)
            prefix = AddressPrefix(prefix_int)
            buf = buf[1:]
        return Address(prefix, buf)

    def to_bytes(self) -> bytes:
        """
        Returns data as bytes from the address object

        :return: :class:`.bytes` data including information of Address object
        """
        prefix_byte = self.prefix.value.to_bytes(1, DATA_BYTE_ORDER)
        return prefix_byte + self.body

    @staticmethod
    def from_prefix_and_int(prefix: AddressPrefix, num: int):
        num_bytes = int_to_bytes(num)
        zero_size = ICON_ADDRESS_BODY_SIZE - len(num_bytes)
        if zero_size < 0:
            raise InvalidParamsException(f'num_bytes is over 20 bytes num: {num}')
        return Address(prefix, b'\x00' * zero_size + num_bytes)


class MalformedAddress(Address):
    """This class only exists to support an invalid format address which was created by legacy bug
    """
    def __init__(self,
                 address_prefix: AddressPrefix,
                 address_body: bytes) -> None:
        """Constructor

        :param address_prefix: address prefix enumerator
        :param address_body: 20-byte address body
        """

        super().__init__(address_prefix, address_body, ignore_length_validate=True)

    @staticmethod
    def from_string(address: str):
        """Create Address object from 42-char address

        :return: :class:`.Address`
        """

        try:
            if address.startswith('hx'):
                body = address[2:]
            else:
                body = address

            address_body = bytes.fromhex(body)
        except:
            raise InvalidParamsException('Invalid address')

        return MalformedAddress(AddressPrefix.EOA, address_body)


# cx0000000000000000000000000000000000000000
SYSTEM_SCORE_ADDRESS = Address.from_prefix_and_int(AddressPrefix.CONTRACT, 0)
ZERO_SCORE_ADDRESS = SYSTEM_SCORE_ADDRESS
# cx0000000000000000000000000000000000000001
GOVERNANCE_SCORE_ADDRESS = Address.from_prefix_and_int(AddressPrefix.CONTRACT, 1)
# A dummy address for handling GETAPI message
GETAPI_DUMMY_ADDRESS = Address.from_data(AddressPrefix.CONTRACT, "SCORE_API".encode())

BUILTIN_SCORE_ADDRESS_MAPPER = {
    'system': SYSTEM_SCORE_ADDRESS,
    'governance': GOVERNANCE_SCORE_ADDRESS,
    'getapi_dummy': GETAPI_DUMMY_ADDRESS,
}
