from iconservice import *

TAG = 'MethodCaller'


class IMethodCaller(InterfaceScore):
    @interface
    def externalDummy(self):
        pass

    @interface
    def externalWrite(self):
        pass

    @interface
    def readonlyReturnInt(self) -> int:
        pass


class MethodCaller(IconScoreBase):

    def __init__(self, db: IconScoreDatabase) -> None:
        super().__init__(db)
        self._var_int = VarDB('var_int', db, value_type=int)

    def on_install(self) -> None:
        super().on_install()
        self.Called("on_install", 0)

    def on_update(self) -> None:
        super().on_update()
        self.Called("on_update", 0)

    @payable
    def fallback(self) -> None:
        self.Called("fallback", 0)

    @eventlog(indexed=2)
    def Called(self, method: str, seq: int):
        pass

    @external
    def externalDummy(self) -> None:
        pass

    @external
    @payable
    def payableDummy(self) -> None:
        pass

    @external
    def externalWriteInt(self, _value: int):
        self._var_int.set(_value)

    @external
    def externalEventLog(self, _value: int):
        self.Called("externalLog", _value)

    @external
    def externalReturnInt(self) -> int:
        self.Called("externalReturnInt", 0)
        return self._var_int.get()

    @external(readonly=True)
    def readonlyWriteInt(self, _value: int) -> int:
        self._var_int.set(_value)
        return _value

    @external(readonly=True)
    def readonlyReturnInt(self) -> int:
        return self._var_int.get()

    @external(readonly=True)
    def readonlyEventLog(self) -> int:
        self.Called("readonlyEventLog", 0)
        return self._var_int.get()

    @external
    def externalCallReadonlyReturnInt(self, addr: Address) -> int:
        self.Called('externalCallReadonlyReturnInt', 0)
        score = self.create_interface_score(addr, IMethodCaller)
        value = score.readonlyReturnInt()
        self.Called('externalCallReadonlyReturnInt', 1)
        return value

    @external(readonly=True)
    def readonlyCallReadonlyReturnInt(self, addr: Address) -> int:
        Logger.debug("readonlyCallReadonlyReturnInt#1", TAG)
        score = self.create_interface_score(addr, IMethodCaller)
        value = score.readonlyReturnInt()
        Logger.debug("readonlyCallReadonlyReturnInt#2", TAG)
        return value

    @external(readonly=True)
    def readonlyCallExternalDummy(self, addr: Address) -> int:
        self.Called('readonlyCallExternalDummy', 0)
        score = self.create_interface_score(addr, IMethodCaller)
        value = score.readonlyReturnInt()
        self.Called('readonlyCallExternalDummy', 2)
        return value
