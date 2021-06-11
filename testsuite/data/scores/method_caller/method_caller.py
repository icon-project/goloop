from iconservice import *

TAG = 'MethodCaller'


def require(condition: bool):
    if not condition:
        revert("Unexpected return value")


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

    @interface
    def readonlyTransfer(self, addr: Address) -> int:
        pass

    @interface
    def readonlyCallReadonlyReturnInt(self, addr: Address) -> int:
        pass

    @interface
    def readonlyCallExternalDummy(self, addr: Address) -> int:
        pass

    @interface
    def fallback(self):
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

    @external
    def callFallback(self, addr: Address):
        score = self.create_interface_score(addr, IMethodCaller)
        score.fallback()

    @eventlog(indexed=2)
    def Called(self, method: str, seq: int):
        pass

    @external
    @payable
    def payableDummy(self) -> None:
        pass

    @external
    def externalDummy(self) -> None:
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

    @external
    def externalCallReadonlyCallReadonlyReturnInt(self, addr: Address) -> int:
        Logger.debug("externalCallReadonlyReturnInt#1", TAG)
        score = self.create_interface_score(addr, IMethodCaller)
        Logger.debug("externalCallReadonlyReturnInt#2", TAG)
        value = score.readonlyCallReadonlyReturnInt(self.address)
        Logger.debug("externalCallReadonlyReturnInt#3", TAG)
        return value

    @external
    def externalCallReadonlyCallExternalDummy(self, addr: Address):
        Logger.debug("externalCallReadonlyCallExternalDummy#1", TAG)
        score = self.create_interface_score(addr, IMethodCaller)
        Logger.debug("externalCallReadonlyCallExternalDummy#2", TAG)
        score.readonlyCallExternalDummy(self.address)
        Logger.debug("externalCallReadonlyCallExternalDummy#3", TAG)

    @external
    def externalCallReadonlyTransfer(self, addr: Address):
        Logger.debug("externalCallReadonlyTransfer#1", TAG)
        score = self.create_interface_score(addr, IMethodCaller)
        Logger.debug("externalCallReadonlyTransfer#2", TAG)
        score.readonlyTransfer(self.address)
        Logger.debug("externalCallReadonlyTransfer#3", TAG)

    @external(readonly=True)
    def readonlyReturnInt(self) -> int:
        return self._var_int.get()

    @external(readonly=True)
    def readonlyWriteInt(self, _value: int) -> int:
        self._var_int.set(_value)
        return _value

    @external(readonly=True)
    def readonlyEventLog(self) -> int:
        self.Called("readonlyEventLog", 0)
        return self._var_int.get()

    @external(readonly=True)
    def readonlyTransfer(self, addr: Address) -> int:
        self.icx.transfer(addr, 1)
        return 0

    @external(readonly=True)
    def readonlyCallReadonlyReturnInt(self, addr: Address) -> int:
        Logger.debug("readonlyCallReadonlyReturnInt#1", TAG)
        score = self.create_interface_score(addr, IMethodCaller)
        value = score.readonlyReturnInt()
        Logger.debug("readonlyCallReadonlyReturnInt#2", TAG)
        return value

    @external(readonly=True)
    def readonlyCallReadonlyTransfer(self, addr: Address) -> int:
        Logger.debug("readonlyCallReadonlyReturnInt#1", TAG)
        score = self.create_interface_score(addr, IMethodCaller)
        value = score.readonlyTransfer(self.tx.origin)
        Logger.debug("readonlyCallReadonlyReturnInt#2", TAG)
        return value

    @external(readonly=True)
    def readonlyCallExternalDummy(self, addr: Address) -> int:
        Logger.debug('readonlyCallExternalDummy#1')
        score = self.create_interface_score(addr, IMethodCaller)
        Logger.debug('readonlyCallExternalDummy#2')
        score.externalDummy()
        Logger.debug('readonlyCallExternalDummy#3')
        return 0

    @external
    def intercallProxy(self, addr: Address):
        score = self.create_interface_score(addr, IMethodCaller)
        value = score.readonlyReturnInt()
        if addr.is_contract:
            require(value == 0)
        else:
            require(value is None)
