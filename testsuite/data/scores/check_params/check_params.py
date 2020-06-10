from iconservice import *

TAG = 'CheckParams'

TYPE_BOOL = 0
TYPE_ADDR = 1
TYPE_INT = 2
TYPE_BYTES = 3
TYPE_STR = 4


class InterCallInterface(InterfaceScore):
    @interface
    def call_bool(self, param: bool):
        pass

    @interface
    def call_address(self, param: Address):
        pass

    @interface
    def call_int(self, param: int):
        pass

    @interface
    def call_bytes(self, param: bytes):
        pass

    @interface
    def call_str(self, param: str):
        pass

    @interface
    def call_default_param(self, default_param: bytes = None) -> str:
        pass

    @interface
    def call_all(self, p_bool: bool, p_addr: Address, p_int: int, p_str: str, p_bytes: bytes):
        pass

    @interface
    def call_all_default(self, _bool: bool, _int: int):
        pass


class CheckParams(IconScoreBase):
    _TYPE = 'types'

    def __init__(self, db: IconScoreDatabase) -> None:
        super().__init__(db)
        self._type_val = DictDB(self._TYPE, db, value_type=str)

    def on_install(self) -> None:
        super().on_install()

    def on_update(self) -> None:
        super().on_update()

    @external
    def call_bool(self, param: bool):
        if param is None:
            self._type_val['bool'] = 'None'
        elif isinstance(param, bool):
            self._type_val['bool'] = str(param).lower()
        else:
            self._type_val['bool'] = "not bool"

    @external
    def call_address(self, param: Address):
        if param is None:
            value = "None"
        elif isinstance(param, Address):
            value = str(param)
            self.LogCallValue(None, None, None, param, None)
        else:
            value = "not address"
        self._type_val['Address'] = value

    @external
    def call_int(self, param: int):
        if param is None:
            value = 'None'
        elif isinstance(param, int):
            value = str(param)
            self.LogCallValue(None, param, None, None, None)
        else:
            value = "not int"
        self._type_val['int'] = value

    @external
    def call_bytes(self, param: bytes):
        if param is None:
            value = "None"
        elif isinstance(param, bytes):
            value = "0x" + param.hex()
            self.LogCallValue(None, None, None, None, param)
        else:
            value = "not bytes"
        self._type_val['bytes'] = value

    @external
    def call_str(self, param: str):
        if param is None:
            value = "None"
        elif isinstance(param, str):
            value = param
            self.LogCallValue(None, None, param, None, None)
        else:
            value = "not str"
        self._type_val['str'] = value

    @external
    def call_all(self, p_bool: bool, p_addr: Address, p_int: int, p_str: str, p_bytes: bytes):
        self._type_val['all'] = "all"
        if not isinstance(p_bool, bool):
            self._type_val['all'] = "not bool"
        elif not isinstance(p_addr, Address):
            self._type_val['all'] = "not Address"
        elif not isinstance(p_int, int):
            self._type_val['all'] = "not int"
        elif not isinstance(p_str, str):
            self._type_val['all'] = "not str"
        elif not isinstance(p_bytes, bytes):
            self._type_val['all'] = "not bytes"

    def convert_type(self, param, ptype):
        if ptype == TYPE_BOOL:
            if isinstance(param, bool):
                o = param
            else:
                o = bool(param)
        elif ptype == TYPE_ADDR:
            if isinstance(param, Address):
                o = param
            else:
                o = Address.from_bytes(b'\x00068e432c41f4de56ad3c')
        elif ptype == TYPE_BYTES:
            if isinstance(param, bytes):
                o = param
            else:
                o = bytes.fromhex('414243')  # 'ABC'
        elif ptype == TYPE_INT:
            if isinstance(param, bool):
                o = int(param)
            elif isinstance(param, int):
                o = param
            else:
                o = int(2)
        elif ptype == TYPE_STR:
            if isinstance(param, str):
                o = param
            else:
                o = str(param)
        else:
            raise Exception(f'IllegalPType{ptype})')
        return o

    @external
    def inter_call_bool(self, _to: Address, param: bool, ptype: int):
        recipient_score = self.create_interface_score(_to, InterCallInterface)
        recipient_score.call_bool(self.convert_type(param, ptype))

    @external
    def inter_call_address(self, _to: Address, param: Address, ptype: int):
        recipient_score = self.create_interface_score(_to, InterCallInterface)
        recipient_score.call_address(self.convert_type(param, ptype))

    @external
    def inter_call_int(self, _to: Address, param: int, ptype: int):
        recipient_score = self.create_interface_score(_to, InterCallInterface)
        recipient_score.call_int(self.convert_type(param, ptype))

    @external
    def inter_call_bytes(self, _to: Address, param: bytes, ptype: int):
        recipient_score = self.create_interface_score(_to, InterCallInterface)
        recipient_score.call_bytes(self.convert_type(param, ptype))

    @external
    def inter_call_str(self, _to: Address, param: str, ptype: int):
        recipient_score = self.create_interface_score(_to, InterCallInterface)
        recipient_score.call_str(self.convert_type(param, ptype))

    @external
    def inter_call_all(self, _to: Address, p_bool: bool, p_addr: Address, p_int: int, p_str: str, p_bytes: bytes):
        recipient_score = self.create_interface_score(_to, InterCallInterface)
        recipient_score.call_all(p_bool, p_addr, p_int, p_str, p_bytes)

    @external(readonly=True)
    def check_bool(self) -> str:
        return self._type_val['bool']

    @external(readonly=True)
    def check_address(self) -> str:
        return self._type_val['Address']

    @external(readonly=True)
    def check_int(self) -> str:
        return self._type_val['int']

    @external(readonly=True)
    def check_bytes(self) -> str:
        return self._type_val['bytes']

    @external(readonly=True)
    def check_str(self) -> str:
        return self._type_val['str']

    @external(readonly=True)
    def check_all(self) -> str:
        return self._type_val['all']

    @external
    def call_default_param(self, default_param: bytes = None):
        if default_param is None:
            self._type_val['default'] = "None"
        elif isinstance(default_param, bytes):
            self._type_val['default'] = default_param.decode()
        else:
            self._type_val['default'] = "not bytes"
        return

    @external(readonly=True)
    def check_default(self) -> str:
        return self._type_val['default']

    @external
    def inter_call_default_param(self, _to: Address):
        recipient_score = self.create_interface_score(_to, InterCallInterface)
        recipient_score.call_default_param()

    @external
    def inter_call_with_none(self, _to: Address, ptype: int):
        recipient_score = self.create_interface_score(_to, InterCallInterface)
        if ptype == TYPE_BOOL:
            recipient_score.call_bool(None)
        elif ptype == TYPE_ADDR:
            recipient_score.call_address(None)
        elif ptype == TYPE_BYTES:
            recipient_score.call_bytes(None)
        elif ptype == TYPE_INT:
            recipient_score.call_int(None)
        elif ptype == TYPE_STR:
            recipient_score.call_str(None)
        else:
            self.revert(f'IllegalPType{ptype})')

    @external
    def inter_call_with_default_param(self, _to: Address, p_bool: bool = True,
                                      p_addr: Address = ZERO_SCORE_ADDRESS,
                                      p_int: int = 0, p_str: str = "", p_bytes: bytes = bytes([0])):
        recipient_score = self.create_interface_score(_to, InterCallInterface)
        recipient_score.call_bool(p_bool)
        recipient_score.call_address(p_addr)
        recipient_score.call_int(p_int)
        recipient_score.call_str(p_str)
        recipient_score.call_bytes(p_bytes)

    @external
    def inter_call_with_more_params(self, _to: Address):
        recipient_score = self.create_interface_score(_to, InterCallInterface)
        recipient_score.call_bytes(bytes.fromhex('414243'), 123)

    @external
    def inter_call_empty_str(self, _to: Address):
        recipient_score = self.create_interface_score(_to, InterCallInterface)
        recipient_score.call_str("")

    @external(readonly=True)
    def check_sender(self) -> Address:
        return self.msg.sender

    @eventlog(indexed=0)
    def LogCallValue(self, _bool: bool, _int: int, _str: str, _addr: Address, _bytes: bytes):
        pass

    @external
    def call_all_default(self, _bool: bool = None, _int: int = None, _str: str = None,
                         _addr: Address = None, _bytes: bytes = None):
        self.LogCallValue(_bool, _int, _str, _addr, _bytes)

    @external
    def inter_call_with_less_params(self, _to: Address, _bool: bool, _int: int):
        score = self.create_interface_score(_to, InterCallInterface)
        score.call_all_default(_bool, _int)
