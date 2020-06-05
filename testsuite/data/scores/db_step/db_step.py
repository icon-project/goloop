from iconservice import *

TAG = 'StepTest'

'''
get, set, set data to 2 var, replace, replaces, delete, deletes, eventLog, api_call
inter_call 
'''
class DbStep(IconScoreBase):
    def __init__(self, db: IconScoreDatabase) -> None:
        super().__init__(db)

        # value_type can be int, str, bytes and Address
        self._var_int = VarDB("var_int", db, value_type=int)
        self._var_str = VarDB("var_str", db, value_type=str)
        self._var_bytes = VarDB("var_bytes", db, value_type=bytes)
        self._var_addr = VarDB("var_addr", db, value_type=Address)

        self._dict_int = DictDB("dict_int", db, value_type=int)
        self._dict_str = DictDB("dict_str", db, value_type=str)
        self._dict_bytes = DictDB("dict_bytes", db, value_type=bytes)
        self._dict_addr = DictDB("dict_addr", db, value_type=Address)

        self._arr_int = ArrayDB("arr_int", db, value_type=int)
        self._arr_str = ArrayDB("arr_str", db, value_type=str)
        self._arr_bytes = ArrayDB("arr_bytes", db, value_type=bytes)
        self._arr_addr = ArrayDB("arr_addr", db, value_type=Address)

    def on_install(self) -> None:
        super().on_install()

    def on_update(self) -> None:
        super().on_update()

    @external(readonly=True)
    def get(self) -> int:
        return self._var_int.get()

    @external
    def set(self, var: int):
        self._var_int.set(var)

    @external
    def setStr(self, var: str):
        self._var_str.set(var)

    @external
    def setStrToDict(self, key: str, val: str):
        self._dict_str[str(key)] = str(val)

    @external
    def setToVar(self, v_int: int, v_str: str = None, v_bytes: bytes = None, v_addr: Address = None):
        if v_addr is not None:
            self._var_addr.set(v_addr)
        elif v_bytes is not None:
            self._var_bytes.set(v_bytes)
        elif v_str is not None:
            self._var_str.set(v_str)
        else:
            self._var_int.set(v_int)

    # type : 0 for int, 1 for str, 2 for bytes, 3 for addr
    @external
    def getFromVar(self, type: int):
        if type == 0:
            self._var_int.get()
        elif type == 1:
            self._var_str.get()
        elif type == 2:
            self._var_bytes.get()
        else:
            self._var_addr.get()

    @external(readonly=True)
    def readFromVar(self, type: int) -> str:
        if type == 0:
            val: int = self._var_int.get()
            return hex(val)
        elif type == 1:
            val: str = self._var_str.get()
            return val
        elif type == 2:
            val: bytes = self._var_bytes.get()
            return "0x"+val.hex()
        else:
            addr: Address = self._var_addr.get()
            return str(addr)

    # type : 0 for int, 1 for str, 2 for bytes, 3 for addr
    @external
    def delFromVar(self, type: int):
        if type == 0:
            self._var_int.remove()
        elif type == 1:
            self._var_str.remove()
        elif type == 2:
            self._var_bytes.remove()
        else:
            self._var_addr.remove()
