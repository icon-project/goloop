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
        if isinstance(param, bool):
            self._type_val['bool'] = str(param).lower()
        else:
            self._type_val['bool'] = "not bool"

    @external
    def call_address(self, param: Address):
        if param is None:
            self._type_val['Address'] = "None"
        elif isinstance(param, Address):
            self._type_val['Address'] = str(param)
        else:
            self._type_val['Address'] = "not address"

    @external
    def call_int(self, param: int):
        if isinstance(param, int):
            self._type_val['int'] = str(param)
        else:
            self._type_val['int'] = "not int"

    @external
    def call_bytes(self, param: bytes):
        if param is None:
            self._type_val['bytes'] = "None"
        elif isinstance(param, bytes):
            self._type_val['bytes'] = str(param[0])
        else:
            self._type_val['bytes'] = "not bytes"

    @external
    def call_str(self, param: str):
        if isinstance(param, str):
            self._type_val['str'] = str(param)
        else:
            self._type_val['str'] = "not str"

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
        o = None
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
                o = bytes(12345)
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
        else:
            self._type_val['default'] = "default"
        return

    @external(readonly=True)
    def check_default(self) -> str:
        return self._type_val['default']

    @external
    def inter_call_default_param(self, _to : Address):
        recipient_score = self.create_interface_score(_to, InterCallInterface)
        recipient_score.call_default_param()

    @external
    def inter_call_with_none(self, _to :Address, ptype: int):
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
            raise Exception(f'IllegalPType{ptype})')

    @external
    def inter_call_with_more_params(self, _to :Address):
        recipient_score = self.create_interface_score(_to, InterCallInterface)
        recipient_score.call_bytes(bytes(12345), 123)

