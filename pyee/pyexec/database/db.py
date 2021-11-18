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

from typing import TYPE_CHECKING, Optional, Tuple

from ..base.exception import DatabaseException, InvalidParamsException
from ..icon_constant import IconScoreContextType, IconScoreFuncType
from ..iconscore.icon_score_context import ContextGetter, ContextContainer
from ..iconscore.icon_score_step import StepType, OutOfStepException
from ..utils import sha3_256

if TYPE_CHECKING:
    from ..iconscore.icon_score_context import IconScoreContext
    from ..base.address import Address


def rlp_encode_bytes(b: bytes) -> bytes:
    blen = len(b)
    if blen == 1 and b[0] < 0x80:
        return b
    elif blen <= 55:
        return bytes([blen + 0x80]) + b
    len_bytes = rlp_get_bytes(blen)
    return bytes([len(len_bytes) + 0x80 + 55]) + len_bytes + b


def rlp_get_bytes(x: int) -> bytes:
    if x == 0:
        return b''
    else:
        return rlp_get_bytes(int(x / 256)) + bytes([x % 256])


def get_encoded_key(bytes_key: bytes) -> bytes:
    return rlp_encode_bytes(bytes_key)


def concat_encoded(*args) -> bytes:
    return b''.join(filter(lambda x: x is not None, args))


def _is_db_writable_on_context(context: 'IconScoreContext'):
    """Check if db is writable on a given context

    :param context:
    :return:
    """
    if context is None:
        return False

    context_type = context.type
    func_type = context.func_type

    return context_type != IconScoreContextType.QUERY and \
        func_type != IconScoreFuncType.READONLY


class DummyDatabase(object):
    def get(self, key: bytes) -> bytes:
        raise DatabaseException('No permission')

    def put(self, key: bytes, value: bytes) -> None:
        raise DatabaseException('No permission')

    def delete(self, key: bytes) -> None:
        raise DatabaseException('No permission')

    def close(self) -> None:
        raise DatabaseException('No permission')


class ProxyDatabase(object):

    def __init__(self, proxy) -> None:
        self._proxy = proxy

    def get(self, key: bytes) -> bytes:
        """Get the value for the specified key.

        :param key: (bytes): key to retrieve
        :return: value for the specified key, or None if not found
        """
        return self._proxy.get_value(key)

    def put(self, key: bytes, value: bytes, cb: callable) -> None:
        """Set a value for the specified key.

        :param key: (bytes): key to set
        :param value: (bytes): data to be stored
        :param cb: handler for old value
        """
        self._proxy.set_value(key, value, cb)

    def delete(self, key: bytes, cb: callable) -> None:
        """Delete the key/value pair for the specified key.

        :param key: key to delete
        :param cb: handler for old value
        """
        self._proxy.set_value(key, None, cb)

    def contains(self, prefix: bytes, value: bytes, limit: int) -> Tuple[bool, int, int]:
        """
        Check whether the value is in the ArrayDB with the prefix

        :param prefix: ArrayDB prefix
        :param value: value to search
        :param limit: available steps
        :return: whether it's in or not, number of value retrieval, size of
                retrieved values
        """
        return self._proxy.contains(prefix, value, limit)

    def close(self) -> None:
        """Close the database.
        """
        if self._proxy:
            self._proxy = None


class ContextDatabase(object):
    """Context-aware database used internally
    """

    def __init__(self, db) -> None:
        """Constructor

        :param db: Proxy database instance
        """
        self._db = db

    def get(self, context: Optional['IconScoreContext'], key: bytes) -> bytes:
        """Get the value for the specified key

        :param context:
        :param key:
        :return: value
        """
        value = self._db.get(key)
        self._charge_step_get(context, value if value is not None else b'')
        return value

    def put(self, context: Optional['IconScoreContext'], key: bytes, value: Optional[bytes]) -> None:
        """Set the value for the specified key

        :param context:
        :param key:
        :param value:
        """
        if not _is_db_writable_on_context(context):
            raise DatabaseException('No permission to write')

        if value:
            try:
                # apply steps for set first
                self._charge_step_set(context, value)
                self._db.put(key, value, self.__refund_handler)
            except OutOfStepException as e:
                # restore to the previous state
                context.step_counter.add_step(e.step_used - e.step_limit)
                # do fallback handling
                old_val = self._db.get(key)
                if context.step_counter.schema == 0:
                    if old_val:
                        context.step_counter.apply_step(StepType.REPLACE, len(value))
                    else:
                        context.step_counter.apply_step(StepType.SET, len(value))
                else:
                    # refund first if old value exists
                    if old_val:
                        self.__refund_handler(True, len(old_val))
                    context.step_counter.apply_step(StepType.SET, len(value))
                self._db.put(key, value, None)
        else:
            if value is None:
                raise DatabaseException('value should not be None')
            self._db.put(key, value, self.__delete_handler)

    def delete(self, context: Optional['IconScoreContext'], key: bytes):
        """Delete the entry for the specified key

        :param context:
        :param key:
        """
        if not _is_db_writable_on_context(context):
            raise DatabaseException('No permission to delete')

        self._db.delete(key, self.__delete_handler)

    def contains(self, context: Optional['IconScoreContext'], prefix: bytes, value: bytes) -> bool:
        """
        Check whether specified value is included or not

        :param context:
        :param prefix: prefix of the ArrayDB
        :param value: value to find
        """
        yn, cnt, size = self._db.contains(prefix, value, context.step_counter.step_remained)
        if cnt > 1:
            base_step = context.step_counter.get_base_step(StepType.GET) * (cnt - 1)
            context.step_counter.consume_step(StepType.GET, base_step)
        context.step_counter.apply_step(StepType.GET, size)
        return yn

    def close(self, context: 'IconScoreContext') -> None:
        """close db

        :param context:
        """
        if not _is_db_writable_on_context(context):
            raise DatabaseException('No permission to close')

    @staticmethod
    def __delete_handler(has_old: bool, old_size: int, new_size: int = 0) -> None:
        """ callback for delete
        """
        context = ContextContainer._get_context()
        if context and context.step_counter and \
                context.type == IconScoreContextType.INVOKE:
            if has_old:
                context.step_counter.apply_step(StepType.DELETE, old_size)

    @staticmethod
    def __refund_handler(has_old: bool, old_size: int, new_size: int = 0) -> None:
        """ callback for refund in case of replace
        """
        context = ContextContainer._get_context()
        if context and context.step_counter and \
                context.type == IconScoreContextType.INVOKE:
            if has_old:
                count = new_size if context.step_counter.schema == 0 else old_size
                context.step_counter.refund_step(count)

    @staticmethod
    def _charge_step_get(context: 'IconScoreContext', value: bytes):
        """ charge steps for get
        """
        if context and context.step_counter:
            context.step_counter.apply_step(StepType.GET, len(value))

    @staticmethod
    def _charge_step_set(context: 'IconScoreContext', value: bytes):
        """ charge steps for set
        """
        if context and context.step_counter and \
                context.type == IconScoreContextType.INVOKE:
            # charge for the new value
            context.step_counter.apply_step(StepType.SET, len(value))


class IconScoreDatabase(ContextGetter):
    """It is used in IconScore

    IconScore can access its states only through IconScoreDatabase
    """
    def __init__(self,
                 address: 'Address',
                 context_db: 'ContextDatabase',
                 tag: bytes = None,
                 prefix: bytes = None) -> None:
        """Constructor

        :param address: the address of SCORE which this db is assigned to
        :param context_db: ContextDatabase
        :param prefix:
        """
        self._address = address
        self._context_db = context_db
        self._tag = tag
        self._prefix = prefix

    @property
    def address(self):
        return self._address

    def get(self, key: Optional[bytes]) -> bytes:
        """
        Gets the value for the specified key

        :param key: key to retrieve
        :return: value for the specified key, or None if not found
        """
        hashed_key = self._hash_key(key)
        return self._context_db.get(self._context, hashed_key)

    def put(self, key: Optional[bytes], value: bytes):
        """
        Sets a value for the specified key.

        :param key: key to set
        :param value: value to set
        """
        hashed_key = self._hash_key(key)
        self._context_db.put(self._context, hashed_key, value)

    def get_sub_db(self, prefix: bytes, tag: bytes = None) -> 'IconScoreDatabase':
        """
        Returns sub db with a prefix

        :param prefix: the prefix used by this sub db.
        :param tag: the type tag used by this sub db.
        :return: sub db
        """
        if prefix is None:
            raise InvalidParamsException('prefix is None')

        prefix = get_encoded_key(prefix)

        if tag is not None and self._tag is None:
            tag = concat_encoded(tag, prefix)
            prefix = None
        else:
            tag = self._tag

        if self._prefix is not None:
            prefix = concat_encoded(self._prefix, prefix)

        icon_score_database = IconScoreDatabase(
            self._address, self._context_db, tag, prefix)

        return icon_score_database

    def delete(self, key: Optional[bytes]):
        """
        Deletes the key/value pair for the specified key.

        :param key: key to delete
        """
        hashed_key = self._hash_key(key)
        self._context_db.delete(self._context, hashed_key)

    def contains(self, value: bytes) -> bool:
        """
        Returns whether the value is included in the ArrayDB

        :param value:
        :return:
        """
        prefix = concat_encoded(self._tag, self._prefix)
        return self._context_db.contains(self._context, prefix, value)

    def close(self):
        self._context_db.close(self._context)

    def _hash_key(self, key: Optional[bytes]) -> bytes:
        """All key are hashed and stored to StateDB

        :params key: key passed by SCORE
        :return: hashed key bytes
        """
        if key is not None:
            key = get_encoded_key(key)
        key = concat_encoded(self._tag, self._prefix, key)
        return sha3_256(key)
