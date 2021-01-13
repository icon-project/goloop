from iconservice import *

TAG = 'ScoreApi'


class Person(TypedDict):
    name: str
    age: int


class ScoreApi(IconScoreBase):

    def __init__(self, db: IconScoreDatabase) -> None:
        super().__init__(db)

    def on_install(self) -> None:
        super().on_install()

    def on_update(self) -> None:
        super().on_update()

    @external
    def externalMethod(self) -> str:
        return "externalMethod"

    @external(readonly=True)
    def externalReadonlyMethod(self) -> str:
        return "externalReadonlyMethod"

    # This should not be exposed to the API list
    @payable
    def payableMethod(self) -> str:
        return "payableMethod"

    @payable
    @external
    def payableExternalMethod(self) -> str:
        return "payableExternalMethod"

    @external
    @payable
    def externalPayableMethod(self) -> str:
        return "externalPayableMethod"

    @external(readonly=False)
    def externalReadonlyFalseMethod(self) -> str:
        return "externalReadonlyFalseMethod"

    # Possible data types for function parameters are int, str, bytes, bool, Address.
    # List and Struct parameters are now newly supported.
    # Return types can be int, str, bytes, bool, Address, list, dict.

    @external(readonly=True)
    def param_int(self, param1: int) -> int:
        return param1

    @external(readonly=True)
    def param_str(self, param1: str) -> str:
        return param1

    @external(readonly=True)
    def param_bytes(self, param1: bytes) -> bytes:
        return param1

    @external(readonly=True)
    def param_bool(self, param1: bool) -> bool:
        return param1

    @external(readonly=True)
    def param_Address(self, param1: Address) -> Address:
        return param1

    @external(readonly=True)
    def param_List(self, param1: List[str]) -> List[str]:
        return param1

    @external(readonly=True)
    def param_ListList(self, param1: List[List[str]]) -> List[List[str]]:
        return param1

    @external(readonly=True)
    def param_Struct(self, param1: Person) -> Person:
        return param1

    @external(readonly=True)
    def param_ListStruct(self, param1: List[Person]) -> List[Person]:
        return param1

    @external(readonly=True)
    def return_list(self, rtype: str) -> list:
        if rtype == "str":
            return ["hello", "world"]
        elif rtype == "bytes":
            hello = bytes([0x68, 0x65, 0x6c, 0x6c, 0x6f])
            world = bytes([0x77, 0x6f, 0x72, 0x6c, 0x64])
            return [hello, world]
        elif rtype == "bool":
            return [True, False]
        elif rtype == "Address":
            return [self.address, self.owner]
        return [0, 1, 100]

    @external(readonly=True)
    def return_dict(self, rtype: str) -> dict:
        if rtype == "str":
            return {"key0": "hello", "key1": "world"}
        elif rtype == "bytes":
            hello = bytes([0x68, 0x65, 0x6c, 0x6c, 0x6f])
            world = bytes([0x77, 0x6f, 0x72, 0x6c, 0x64])
            return {"key0": hello, "key1": world}
        elif rtype == "bool":
            return {"key0": True, "key1": False}
        elif rtype == "Address":
            return {"key0": self.address, "key1": self.owner}
        return {"key0": 0, "key1": 1, "key2": 100}

    @payable
    def fallback(self):
        Logger.debug("fallback", TAG)

    @eventlog(indexed=1)
    def eventlog_index1(self, param1: int, param2: str):
        pass
