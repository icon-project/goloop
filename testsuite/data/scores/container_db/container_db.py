from iconservice import *


class ContainerDB(IconScoreBase):
    def __init__(self, db: IconScoreDatabase) -> None:
        super().__init__(db)

        # value_type can be int, str, bytes, bool and Address
        self._var_int = VarDB("var_int", db, value_type=int)
        self._var_str = VarDB("var_str", db, value_type=str)
        self._var_bytes = VarDB("var_bytes", db, value_type=bytes)
        self._var_bool = VarDB("var_bool", db, value_type=bool)
        self._var_addr = VarDB("var_addr", db, value_type=Address)

        self._dict_int = DictDB("dict_int", db, value_type=int)
        self._dict_str = DictDB("dict_str", db, value_type=str)
        self._dict_bytes = DictDB("dict_bytes", db, value_type=bytes)
        self._dict_bool = DictDB("dict_bool", db, value_type=bool)
        self._dict_addr = DictDB("dict_addr", db, value_type=Address)

        self._arr_int = ArrayDB("arr_int", db, value_type=int)
        self._arr_str = ArrayDB("arr_str", db, value_type=str)
        self._arr_bytes = ArrayDB("arr_bytes", db, value_type=bytes)
        self._arr_bool = ArrayDB("arr_bool", db, value_type=bool)
        self._arr_addr = ArrayDB("arr_addr", db, value_type=Address)

    def on_install(self) -> None:
        super().on_install()

    def on_update(self) -> None:
        super().on_update()

    @external(readonly=True)
    def getVar(self, type: str) -> dict:
        if type == 'int':
            return {type: self._var_int.get()}
        elif type == 'str':
            return {type: self._var_str.get()}
        elif type == 'bytes':
            return {type: self._var_bytes.get()}
        elif type == 'bool':
            return {type: self._var_bool.get()}
        elif type == 'addr':
            return {type: self._var_addr.get()}
        return {'error': 'InvalidType'}

    @external
    def setVar(self, v_int: int = None, v_str: str = None, v_bytes: bytes = None,
               v_bool: bool = None, v_addr: Address = None):
        if v_int is not None:
            self._var_int.set(v_int)
        elif v_str is not None:
            self._var_str.set(v_str)
        elif v_bytes is not None:
            self._var_bytes.set(v_bytes)
        elif v_bool is not None:
            self._var_bool.set(v_bool)
        elif v_addr is not None:
            self._var_addr.set(v_addr)

    @external(readonly=True)
    def getDict(self, key: str, type: str) -> dict:
        if type == 'int':
            return {key: self._dict_int[key]}
        elif type == 'str':
            return {key: self._dict_str[key]}
        elif type == 'bytes':
            return {key: self._dict_bytes[key]}
        elif type == 'bool':
            return {key: self._dict_bool[key]}
        elif type == 'addr':
            return {key: self._dict_addr[key]}
        return {'error': 'InvalidType'}

    @external
    def setDict(self, key: str, v_int: int = None, v_str: str = None, v_bytes: bytes = None,
                v_bool: bool = None, v_addr: Address = None):
        if v_int is not None:
            self._dict_int[key] = v_int
        elif v_str is not None:
            self._dict_str[key] = v_str
        elif v_bytes is not None:
            self._dict_bytes[key] = v_bytes
        elif v_bool is not None:
            self._dict_bool[key] = v_bool
        elif v_addr is not None:
            self._dict_addr[key] = v_addr

    @external(readonly=True)
    def getArray(self, type: str) -> list:
        if type == 'int':
            return [item for item in self._arr_int]
        elif type == 'str':
            return [item for item in self._arr_str]
        elif type == 'bytes':
            return [item for item in self._arr_bytes]
        elif type == 'bool':
            return [item for item in self._arr_bool]
        elif type == 'addr':
            return [item for item in self._arr_addr]
        return ['InvalidType']

    @external
    def setArray(self, v_int: int = None, v_str: str = None, v_bytes: bytes = None,
                 v_bool: bool = None, v_addr: Address = None):
        if v_int is not None:
            self._arr_int.put(v_int)
        elif v_str is not None:
            self._arr_str.put(v_str)
        elif v_bytes is not None:
            self._arr_bytes.put(v_bytes)
        elif v_bool is not None:
            self._arr_bool.put(v_bool)
        elif v_addr is not None:
            self._arr_addr.put(v_addr)
