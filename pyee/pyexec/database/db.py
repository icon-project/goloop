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

from typing import TYPE_CHECKING, Optional

from ..base.exception import DatabaseException, InvalidParamsException
from ..icon_constant import IconScoreContextType, IconScoreFuncType
from ..iconscore.icon_score_context import ContextGetter
from ..iconscore.icon_score_step import StepType
from ..utils import sha3_256

if TYPE_CHECKING:
    from ..iconscore.icon_score_context import IconScoreContext
    from ..base.address import Address


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

    def put(self, key: bytes, value: bytes) -> None:
        """Set a value for the specified key.

        :param key: (bytes): key to set
        :param value: (bytes): data to be stored
        """
        self._proxy.set_value(key, value)

    def delete(self, key: bytes) -> None:
        """Delete the key/value pair for the specified key.

        :param key: key to delete
        """
        self._proxy.set_value(key, None)

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
        self.__on_db_get(context, key, value)
        return value

    def put(self, context: Optional['IconScoreContext'], key: bytes, value: Optional[bytes]) -> None:
        """Set the value for the specified key

        :param context:
        :param key:
        :param value:
        """
        if not _is_db_writable_on_context(context):
            raise DatabaseException('No permission to write')

        old_value = self._db.get(key)
        if value:
            self.__on_db_put(context, key, old_value, value)
        elif old_value:
            # If new value is None, then deletes the field
            self.__on_db_delete(context, key, old_value)
        self._db.put(key, value)

    def delete(self, context: Optional['IconScoreContext'], key: bytes):
        """Delete the entry for the specified key

        :param context:
        :param key:
        """
        if not _is_db_writable_on_context(context):
            raise DatabaseException('No permission to delete')

        old_value = self._db.get(key)
        # If old value is None, won't fire the callback
        if old_value:
            self.__on_db_delete(context, key, old_value)
        self._db.delete(key)

    def close(self, context: 'IconScoreContext') -> None:
        """close db

        :param context:
        """
        if not _is_db_writable_on_context(context):
            raise DatabaseException('No permission to close')

    @staticmethod
    def __on_db_get(context: 'IconScoreContext',
                    key: bytes,
                    value: bytes):
        """Invoked when `get` is called in `ContextDatabase`.

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

        :param context: SCORE context
        :param key: key
        :param old_value: old value
        :param new_value: new value
        """
        if context and context.step_counter and \
                context.type == IconScoreContextType.INVOKE:
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

        :param context: SCORE context
        :param key: key
        :param old_value: old value
        """
        if context and context.step_counter and \
                context.type == IconScoreContextType.INVOKE:
            context.step_counter.apply_step(
                StepType.DELETE, len(old_value))


class IconScoreDatabase(ContextGetter):
    """It is used in IconScore

    IconScore can access its states only through IconScoreDatabase
    """
    def __init__(self,
                 address: 'Address',
                 context_db: 'ContextDatabase',
                 prefix: bytes = None) -> None:
        """Constructor

        :param address: the address of SCORE which this db is assigned to
        :param context_db: ContextDatabase
        :param prefix:
        """
        self._address = address
        self._context_db = context_db
        self._prefix = prefix

    @property
    def address(self):
        return self._address

    def get(self, key: bytes) -> bytes:
        """
        Gets the value for the specified key

        :param key: key to retrieve
        :return: value for the specified key, or None if not found
        """
        hashed_key = self._hash_key(key)
        return self._context_db.get(self._context, hashed_key)

    def put(self, key: bytes, value: bytes):
        """
        Sets a value for the specified key.

        :param key: key to set
        :param value: value to set
        """
        hashed_key = self._hash_key(key)
        self._context_db.put(self._context, hashed_key, value)

    @property
    def prefix(self) -> bytes:
        return self._prefix

    def get_sub_db(self, prefix: bytes) -> 'IconScoreDatabase':
        """
        Returns sub db with a prefix

        :param prefix: The prefix used by this sub db.
        :return: sub db
        """
        if prefix is None:
            raise InvalidParamsException('prefix is None')

        if self._prefix is not None:
            prefix = b''.join([self._prefix, prefix])

        icon_score_database = IconScoreDatabase(
            self._address, self._context_db, prefix)

        return icon_score_database

    def delete(self, key: bytes):
        """
        Deletes the key/value pair for the specified key.

        :param key: key to delete
        """
        hashed_key = self._hash_key(key)
        self._context_db.delete(self._context, hashed_key)

    def close(self):
        self._context_db.close(self._context)

    def _hash_key(self, key: bytes) -> bytes:
        """All key are hashed and stored to StateDB

        :params key: key passed by SCORE
        :return: hashed key bytes
        """
        if self._prefix is not None:
            key = self._prefix + key
        return sha3_256(key)
