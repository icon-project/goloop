from iconservice import *

TAG = 'ScoreApi'


class ScoreApi(IconScoreBase):

    def __init__(self, db: IconScoreDatabase) -> None:
        super().__init__(db)

    def on_install(self) -> None:
        super().on_install()

    def on_update(self) -> None:
        super().on_update()

    # readonly, external, payable, isolated
    @external
    def externalMethod(self) -> str:
        return "externalMethod"

    @external(readonly=True)
    def externalReadonlyMethod(self) -> str:
        return "externalReadonlyMethod"

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
    # List and Dict type parameters are not supported yet.
    # Returning types can be int, str, bytes, bool, Address, List, Dict.

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
    def return_list(self) -> list:
        return [1]

    @external(readonly=True)
    def return_dict(self) -> dict:
        return {"1": "1"}

    @payable
    def fallback(self):
        Logger.debug("fallback", TAG)

    @eventlog(indexed=1)
    def eventlog_index1(self, param1: int, param2: str):
        pass
