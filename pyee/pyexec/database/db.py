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
from ..icon_constant import ICON_DB_LOG_TAG, IconScoreContextType, IconScoreFuncType
from ..iconscore.icon_score_context import ContextGetter
from ..logger import Logger
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
        return b'\x00'

    def put(self, key: bytes, value: bytes) -> None:
        pass

    def delete(self, key: bytes) -> None:
        pass

    def close(self) -> None:
        pass


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


class DatabaseObserver(object):
    """ An abstract class of database observer.
    """

    def __init__(self,
                 get_func: callable, put_func: callable, delete_func: callable):
        self.__get_func = get_func
        self.__put_func = put_func
        self.__delete_func = delete_func

    def on_get(self, context: 'IconScoreContext', key: bytes, value: bytes):
        """
        Invoked when `get` is called in `ContextDatabase`

        :param context: SCORE context
        :param key: key
        :param value: value
        """
        if not self.__get_func:
            Logger.warning('__get_func is None', ICON_DB_LOG_TAG)
        self.__get_func(context, key, value)

    def on_put(self,
               context: 'IconScoreContext',
               key: bytes,
               old_value: bytes,
               new_value: bytes):
        """Invoked when `put` is called in `ContextDatabase`.

        :param context: SCORE context
        :param key: key
        :param old_value: old value
        :param new_value: new value
        """
        if not self.__put_func:
            Logger.warning('__put_func is None', ICON_DB_LOG_TAG)
        self.__put_func(context, key, old_value, new_value)

    def on_delete(self,
                  context: 'IconScoreContext',
                  key: bytes,
                  old_value: bytes):
        """Invoked when `delete` is called in `ContextDatabase`.


        :param context: SCORE context
        :param key: key
        :param old_value:
        """
        if not self.__delete_func:
            Logger.warning('__delete_func is None', ICON_DB_LOG_TAG)
        self.__delete_func(context, key, old_value)


class ContextDatabase(object):
    """Database for an IconScore only used in the inside of ICON Service.

    IconScore cannot access this database directly.
    Cache + LevelDB
    """

    def __init__(self, db, is_shared: bool=False) -> None:
        """Constructor

        :param db: KeyValueDatabase instance
        :param is_shared: True if this is shared with all SCOREs
        """
        self.key_value_db = db
        self._is_shared = is_shared

    def get(self, context: Optional['IconScoreContext'], key: bytes) -> bytes:
        """Returns value indicated by key from batch or StateDB

        :param context:
        :param key:
        :return: value
        """
        return self.key_value_db.get(key)

    def put(self,
            context: Optional['IconScoreContext'],
            key: bytes,
            value: Optional[bytes]) -> None:
        """Set the value to StateDB or cache it according to context type

        :param context:
        :param key:
        :param value:
        """
        if not _is_db_writable_on_context(context):
            raise DatabaseException('No permission to write')

        self.key_value_db.put(key, value)

    def delete(self, context: Optional['IconScoreContext'], key: bytes):
        """Delete key from db

        :param context:
        :param key: key to delete from db
        """
        if not _is_db_writable_on_context(context):
            raise DatabaseException('No permission to delete')

        self.key_value_db.delete(key)

    def close(self, context: 'IconScoreContext') -> None:
        """close db

        :param context:
        """
        if not _is_db_writable_on_context(context):
            raise DatabaseException('No permission to close')

        if not self._is_shared:
            return self.key_value_db.close()


class IconScoreDatabase(ContextGetter):
    """It is used in IconScore

    IconScore can access its states only through IconScoreDatabase
    """
    def __init__(self,
                 address: 'Address',
                 context_db: 'ContextDatabase',
                 prefix: bytes=None) -> None:
        """Constructor

        :param address: the address of SCORE which this db is assigned to
        :param context_db: ContextDatabase
        :param prefix:
        """
        self.address = address
        self._prefix = prefix
        self._context_db = context_db
        self._observer: DatabaseObserver = None

    def get(self, key: bytes) -> bytes:
        """
        Gets the value for the specified key

        :param key: key to retrieve
        :return: value for the specified key, or None if not found
        """
        hashed_key = self._hash_key(key)
        value = self._context_db.get(self._context, hashed_key)
        if self._observer:
            self._observer.on_get(self._context, key, value)
        return value

    def put(self, key: bytes, value: bytes):
        """
        Sets a value for the specified key.

        :param key: key to set
        :param value: value to set
        """
        hashed_key = self._hash_key(key)
        if self._observer:
            old_value = self._context_db.get(self._context, hashed_key)
            if value:
                self._observer.on_put(self._context, key, old_value, value)
            elif old_value:
                # If new value is None, then deletes the field
                self._observer.on_delete(self._context, key, old_value)
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
            raise InvalidParamsException(
                'Invalid params: '
                'prefix is None in IconScoreDatabase.get_sub_db()')

        if self._prefix is not None:
            prefix = b''.join([self._prefix, prefix])

        icon_score_database = IconScoreDatabase(
            self.address, self._context_db, prefix)

        icon_score_database.set_observer(self._observer)

        return icon_score_database

    def delete(self, key: bytes):
        """
        Deletes the key/value pair for the specified key.

        :param key: key to delete
        """
        hashed_key = self._hash_key(key)
        if self._observer:
            old_value = self._context_db.get(self._context, hashed_key)
            # If old value is None, won't fire the callback
            if old_value:
                self._observer.on_delete(self._context, key, old_value)
        self._context_db.delete(self._context, hashed_key)

    def close(self):
        self._context_db.close(self._context)

    def set_observer(self, observer: 'DatabaseObserver'):
        self._observer = observer

    def _hash_key(self, key: bytes) -> bytes:
        """All key is hashed and stored
        to StateDB to avoid key conflicts among SCOREs

        :params key: key passed by SCORE
        :return: key bytes
        """
        if self._prefix is not None:
            key = self._prefix + key
        return sha3_256(key)
